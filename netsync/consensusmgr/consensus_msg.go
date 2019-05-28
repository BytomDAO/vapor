package consensusmgr

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/tendermint/go-wire"

	"github.com/vapor/netsync/peers"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

//Consensus msg byte
const (
	BlockSignatureByte = byte(0x10)
	BlockProposeByte   = byte(0x11)
)

//BlockchainMessage is a generic message for this reactor.
type ConsensusMessage interface {
	String() string
	BroadcastMarkSendRecord(ps *peers.PeerSet, peers []string)
	BroadcastFilterTargetPeers(ps *peers.PeerSet) []string
}

var _ = wire.RegisterInterface(
	struct{ ConsensusMessage }{},
	wire.ConcreteType{&BlockSignatureMsg{}, BlockSignatureByte},
	wire.ConcreteType{&BlockProposeMsg{}, BlockProposeByte},
)

//decodeMessage decode msg
func decodeMessage(bz []byte) (msgType byte, msg ConsensusMessage, err error) {
	msgType = bz[0]
	n := int(0)
	r := bytes.NewReader(bz)
	msg = wire.ReadBinary(struct{ ConsensusMessage }{}, r, maxBlockchainResponseSize, &n, &err).(struct{ ConsensusMessage }).ConsensusMessage
	if err != nil && n != len(bz) {
		err = errors.New("DecodeMessage() had bytes left over")
	}
	return
}

type BlockSignatureMsg struct {
	BlockHash [32]byte
	Height    uint64
	Signature []byte
	PubKey    [32]byte
}

//NewBlockSignatureMessage construct new block signature msg
func NewBlockSignatureMsg(blockHash bc.Hash, height uint64, signature []byte, pubKey [32]byte) ConsensusMessage {
	hash := blockHash.Byte32()
	return &BlockSignatureMsg{BlockHash: hash, Height: height, Signature: signature, PubKey: pubKey}
}

func (bs *BlockSignatureMsg) String() string {
	return fmt.Sprintf("{block_hash: %s,block_height:%d,signature:%s,pubkey:%s}", hex.EncodeToString(bs.BlockHash[:]), bs.Height, hex.EncodeToString(bs.Signature), hex.EncodeToString(bs.PubKey[:]))
}

func (bs *BlockSignatureMsg) BroadcastMarkSendRecord(ps *peers.PeerSet, peers []string) {
	for _, peer := range peers {
		ps.MarkBlockSignature(peer, bs.Signature)
	}
}

func (bs *BlockSignatureMsg) BroadcastFilterTargetPeers(ps *peers.PeerSet) []string {
	return ps.PeersWithoutSign(bs.Signature)
}

type BlockProposeMsg struct {
	RawBlock []byte
}

//NewBlockProposeMsg construct new block propose msg
func NewBlockProposeMsg(block *types.Block) (ConsensusMessage, error) {
	rawBlock, err := block.MarshalText()
	if err != nil {
		return nil, err
	}
	return &BlockProposeMsg{RawBlock: rawBlock}, nil
}

//GetProposeBlock get propose block from msg
func (bp *BlockProposeMsg) GetProposeBlock() (*types.Block, error) {
	block := &types.Block{}
	if err := block.UnmarshalText(bp.RawBlock); err != nil {
		return nil, err
	}
	return block, nil
}

func (bp *BlockProposeMsg) String() string {
	block, err := bp.GetProposeBlock()
	if err != nil {
		return "{err: wrong message}"
	}
	blockHash := block.Hash()
	return fmt.Sprintf("{block_height: %d, block_hash: %s}", block.Height, blockHash.String())
}

func (bp *BlockProposeMsg) BroadcastMarkSendRecord(ps *peers.PeerSet, peers []string) {
	block, err := bp.GetProposeBlock()
	if err != nil {
		return
	}

	hash := block.Hash()
	height := block.Height
	for _, peer := range peers {
		ps.MarkBlock(peer, &hash)
		ps.MarkStatus(peer, height)
	}
}

func (bp *BlockProposeMsg) BroadcastFilterTargetPeers(ps *peers.PeerSet) []string {
	block, err := bp.GetProposeBlock()
	if err != nil {
		return nil
	}

	return ps.PeersWithoutBlock(block.Hash())
}
