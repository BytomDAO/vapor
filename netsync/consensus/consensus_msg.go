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
	BlockSignByte    = byte(0x10)
	BlockProposeByte = byte(0x11)
)

//BlockchainMessage is a generic message for this reactor.
type ConsensusMessage interface {
	String() string
}

var _ = wire.RegisterInterface(
	struct{ ConsensusMessage }{},
	wire.ConcreteType{&BlockSignMessage{}, BlockSignByte},
	wire.ConcreteType{&BlockProposeMessage{}, BlockProposeByte},
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

type BlockSignMessage struct {
	BlockID [32]byte
	Height  uint64
	Sign    []byte
	Pubkey  []byte
}

//NewBlockSignMessage construct new mined block msg
func NewBlockSignMessage(blockID [32]byte, height uint64, sign []byte, pubkey []byte) *BlockSignMessage {
	return &BlockSignMessage{BlockID: blockID, Height: height, Sign: sign, Pubkey: pubkey}
}

func (bs *BlockSignMessage) String() string {
	return fmt.Sprintf("{block_hash: %s,block_height: %d,sign:%s,pubkey:%s}", hex.EncodeToString(bs.BlockID[:]), bs.Height, hex.EncodeToString(bs.Sign), hex.EncodeToString(bs.Pubkey))
}

type BlockSignBroadcastMsg struct {
	sign      []byte
	msg       *BlockSignMessage
	transChan byte
}

func NewBlockSignBroadcastMsg(blockID [32]byte, height uint64, sign []byte, pubkey []byte, transChan byte) *BlockSignBroadcastMsg {
	msg := NewBlockSignMessage(blockID, height, sign, pubkey)
	return &BlockSignBroadcastMsg{sign: sign, msg: msg, transChan: transChan}
}

func (m *BlockSignBroadcastMsg) GetChan() byte {
	return m.transChan
}

func (m *BlockSignBroadcastMsg) GetMsg() interface{} {
	return struct{ ConsensusMessage }{m.msg}
}

func (m *BlockSignBroadcastMsg) MsgString() string {
	return m.msg.String()
}

func (m *BlockSignBroadcastMsg) MarkSendRecord(ps *peers.PeerSet, peers []string) {
	for _, peer := range peers {
		ps.MarkBlockSign(peer, m.sign)
	}
}

func (m *BlockSignBroadcastMsg) FilterTargetPeers(ps *peers.PeerSet) []string {
	//TODO: SPV NODE FILTER
	return ps.PeersWithoutSign(m.sign)
}

type BlockProposeMessage struct {
	RawBlock []byte
}

//NewBlockProposeMessage construct new mined block msg
func NewBlockProposeMessage(block *types.Block) (*BlockProposeMessage, error) {
	rawBlock, err := block.MarshalText()
	if err != nil {
		return nil, err
	}
	return &BlockProposeMessage{RawBlock: rawBlock}, nil
}

//GetMineBlock get mine block from msg
func (m *BlockProposeMessage) GetProposeBlock() (*types.Block, error) {
	block := &types.Block{}
	if err := block.UnmarshalText(m.RawBlock); err != nil {
		return nil, err
	}
	return block, nil
}

func (bp *BlockProposeMessage) String() string {
	block, err := bp.GetProposeBlock()
	if err != nil {
		return "{err: wrong message}"
	}
	blockHash := block.Hash()
	return fmt.Sprintf("{block_height: %d, block_hash: %s}", block.Height, blockHash.String())
}

type BlockProposeBroadcastMsg struct {
	block     *types.Block
	msg       *BlockProposeMessage
	transChan byte
}

func NewBlockProposeBroadcastMsg(block *types.Block, transChan byte) (*BlockProposeBroadcastMsg, error) {
	msg, err := NewBlockProposeMessage(block)
	if err != nil {
		return nil, err
	}
	return &BlockProposeBroadcastMsg{block: block, msg: msg, transChan: transChan}, nil
}

func (m *BlockProposeBroadcastMsg) GetChan() byte {
	return m.transChan
}

func (m *BlockProposeBroadcastMsg) GetMsg() interface{} {
	return struct{ ConsensusMessage }{m.msg}
}

func (m *BlockProposeBroadcastMsg) MsgString() string {
	return m.msg.String()
}

func (m *BlockProposeBroadcastMsg) MarkSendRecord(ps *peers.PeerSet, peers []string) {
	hash := m.block.Hash()
	height := m.block.Height
	for _, peer := range peers {
		ps.MarkBlock(peer, &hash)
		ps.MarkStatus(peer, height)
	}
}

func (m *BlockProposeBroadcastMsg) FilterTargetPeers(ps *peers.PeerSet) []string {
	//TODO: SPV NODE FILTER
	return ps.PeersWithoutBlock(m.block.Hash())
}
