package consensus

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/tendermint/go-wire"

	"github.com/vapor/netsync/peers"
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
	BlockID   [32]byte
	Height    uint64
	Signature []byte
	PeerID    [32]byte
}

//NewBlockSignatureMessage construct new mined block msg
func NewBlockSignatureMsg(blockID [32]byte, height uint64, signature []byte, peerId [32]byte) *BlockSignatureMsg {
	return &BlockSignatureMsg{BlockID: blockID, Height: height, Signature: signature, PeerID: peerId}
}

func (bs *BlockSignatureMsg) String() string {
	return fmt.Sprintf("{block_hash: %s,block_height: %d,signature:%s,peerID:%s}", hex.EncodeToString(bs.BlockID[:]), bs.Height, hex.EncodeToString(bs.Signature), hex.EncodeToString(bs.PeerID[:]))
}

type SignatureBroadcastMsg struct {
	signature []byte
	msg       *BlockSignatureMsg
	transChan byte
}

func NewSignatureBroadcastMsg(blockID [32]byte, height uint64, signature []byte, pubkey [32]byte, transChan byte) *SignatureBroadcastMsg {
	msg := NewBlockSignatureMsg(blockID, height, signature, pubkey)
	return &SignatureBroadcastMsg{signature: signature, msg: msg, transChan: transChan}
}

func (s *SignatureBroadcastMsg) GetChan() byte {
	return s.transChan
}

func (s *SignatureBroadcastMsg) GetMsg() interface{} {
	return struct{ ConsensusMessage }{s.msg}
}

func (s *SignatureBroadcastMsg) MsgString() string {
	return s.msg.String()
}

func (s *SignatureBroadcastMsg) MarkSendRecord(ps *peers.PeerSet, peers []string) {
	for _, peer := range peers {
		ps.MarkBlockSignature(peer, s.signature)
	}
}

func (m *SignatureBroadcastMsg) FilterTargetPeers(ps *peers.PeerSet) []string {
	return ps.PeersWithoutSign(m.signature)
}

type BlockProposeMsg struct {
	RawBlock []byte
}

//NewBlockProposeMsg construct new block propose msg
func NewBlockProposeMsg(block *types.Block) (*BlockProposeMsg, error) {
	rawBlock, err := block.MarshalText()
	if err != nil {
		return nil, err
	}
	return &BlockProposeMsg{RawBlock: rawBlock}, nil
}

//GetProposeBlock get propose block from msg
func (m *BlockProposeMsg) GetProposeBlock() (*types.Block, error) {
	block := &types.Block{}
	if err := block.UnmarshalText(m.RawBlock); err != nil {
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

type ProposeBroadcastMsg struct {
	block     *types.Block
	msg       *BlockProposeMsg
	transChan byte
}

func NewBlockProposeBroadcastMsg(block *types.Block, transChan byte) (*ProposeBroadcastMsg, error) {
	msg, err := NewBlockProposeMsg(block)
	if err != nil {
		return nil, err
	}
	return &ProposeBroadcastMsg{block: block, msg: msg, transChan: transChan}, nil
}

func (p *ProposeBroadcastMsg) GetChan() byte {
	return p.transChan
}

func (p *ProposeBroadcastMsg) GetMsg() interface{} {
	return struct{ ConsensusMessage }{p.msg}
}

func (p *ProposeBroadcastMsg) MsgString() string {
	return p.msg.String()
}

func (p *ProposeBroadcastMsg) MarkSendRecord(ps *peers.PeerSet, peers []string) {
	hash := p.block.Hash()
	height := p.block.Height
	for _, peer := range peers {
		ps.MarkBlock(peer, &hash)
		ps.MarkStatus(peer, height)
	}
}

func (p *ProposeBroadcastMsg) FilterTargetPeers(ps *peers.PeerSet) []string {
	return ps.PeersWithoutBlock(p.block.Hash())
}
