package consensusmgr

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/tendermint/go-wire"

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

