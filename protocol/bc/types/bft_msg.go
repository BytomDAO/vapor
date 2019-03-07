package types

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/vapor/common"
	"github.com/vapor/crypto"
	"github.com/vapor/encoding/blockchain"
	"github.com/vapor/protocol/bc"
)

type BftMsgType uint8

const (
	BftPreprepareMessage BftMsgType = iota
	BftPrepareMessage
	BftCommitMessage
)

func (msg BftMsgType) String() string {
	switch msg {
	case BftPreprepareMessage:
		return "BftPreprepareMessage"
	case BftPrepareMessage:
		return "BftPrepareMessage"
	case BftCommitMessage:
		return "BftCommitMessage"
	default:
		return "Unknown bft message type"
	}
}

type BftMsg struct {
	BftType BftMsgType
	Msg     ConsensusMsg
}

type ConsensusMsg interface {
	Type() BftMsgType
	GetRound() uint32
	GetBlockHeight() uint64
	Hash() (hash common.Hash, err error)
}

type PreprepareMsg struct {
	Round uint32
	Block *Block
}

func (msg *PreprepareMsg) Type() BftMsgType {
	return BftPreprepareMessage
}
func (msg *PreprepareMsg) GetBlockHeight() uint64 {
	return msg.Block.Height
}

func (msg *PreprepareMsg) GetRound() uint32 {
	return msg.Round
}

type PreprepareMsgHash struct {
	Round uint32  `json:"round"`
	Hash  bc.Hash `json:"hash"`
}

func (msg *PreprepareMsg) Hash() (hash common.Hash, err error) {

	h := &PreprepareMsgHash{
		Round: msg.Round,
		Hash:  msg.Block.Hash(),
	}

	data, err := json.Marshal(h)
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Sha3Hash(data), nil
}

type PrepareMsg struct {
	Round       uint32
	PrepareAddr string
	BlockHeight uint64
	BlockHash   bc.Hash
	PrepareSig  []byte
}

func (msg *PrepareMsg) Type() BftMsgType {
	return BftPrepareMessage
}

func (msg *PrepareMsg) GetBlockHeight() uint64 {
	return msg.BlockHeight
}

func (msg *PrepareMsg) GetRound() uint32 {
	return msg.Round
}

type PrepareMsgHash struct {
	MsgType     uint32  `json:"msg_type"`
	Round       uint32  `json:"round"`
	PrepareAddr string  `json:"prepare_addr"`
	BlockHeight uint64  `json:"block_height"`
	BlockHash   bc.Hash `json:"block_hash"`
}

func (msg *PrepareMsg) Hash() (hash common.Hash, err error) {

	h := &PrepareMsgHash{
		MsgType:     uint32(BftPrepareMessage),
		Round:       msg.Round,
		PrepareAddr: msg.PrepareAddr,
		BlockHeight: msg.BlockHeight,
		BlockHash:   msg.BlockHash,
	}

	data, err := json.Marshal(h)
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Sha3Hash(data), nil
}

type CommitMsg struct {
	Round       uint32
	Commiter    string
	BlockHeight uint64
	BlockHash   bc.Hash
	CommitSig   []byte
}

func (msg *CommitMsg) Type() BftMsgType {
	return BftCommitMessage
}

func (msg *CommitMsg) GetBlockHeight() uint64 {
	return msg.BlockHeight
}

func (msg *CommitMsg) GetRound() uint32 {
	return msg.Round
}

func (msg *CommitMsg) readFrom(r *blockchain.Reader) (err error) {
	commitMsgByte := []byte{}
	if commitMsgByte, err = blockchain.ReadVarstr31(r); err != nil {
		return err
	}

	if msg.CommitSig, err = blockchain.ReadVarstr31(r); err != nil {
		return err
	}

	h := &CommitMsgHash{}

	if err := json.Unmarshal(commitMsgByte, h); err != nil {
		return err
	}
	msg.Round = h.Round
	msg.Commiter = h.Commiter
	msg.BlockHeight = h.BlockHeight
	msg.BlockHash = h.BlockHash
	return nil
}

func (msg *CommitMsg) writeTo(w io.Writer) error {
	h := &CommitMsgHash{
		MsgType:     uint32(BftCommitMessage),
		Round:       msg.Round,
		Commiter:    msg.Commiter,
		BlockHeight: msg.BlockHeight,
		BlockHash:   msg.BlockHash,
	}

	data, err := json.Marshal(h)
	if err != nil {
		return err
	}

	if _, err := blockchain.WriteVarstr31(w, data); err != nil {
		return err
	}

	if _, err := blockchain.WriteVarstr31(w, msg.CommitSig); err != nil {
		return err
	}

	return nil
}

type CommitMsgHash struct {
	MsgType     uint32  `json:"msg_type"`
	Round       uint32  `json:"round"`
	Commiter    string  `json:"commiter"`
	BlockHeight uint64  `json:"block_height"`
	BlockHash   bc.Hash `json:"block_hash"`
}

func (msg *CommitMsg) Hash() (hash common.Hash, err error) {

	h := &CommitMsgHash{
		MsgType:     uint32(BftCommitMessage),
		Round:       msg.Round,
		Commiter:    msg.Commiter,
		BlockHeight: msg.BlockHeight,
		BlockHash:   msg.BlockHash,
	}

	data, err := json.Marshal(h)
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Sha3Hash(data), nil
}

// Size returns the approximate memory used by all internal contents.
func (msg *CommitMsg) Size() int {
	//return len(msg.Commiter) + len(msg.CommitSig) + common.HashLength + msg.BlockNumber.BitLen()/8
	return 0
}

func (msg *CommitMsg) Dump() {

	fmt.Println("----------------- Dump Commit Message -----------------")

	fmt.Printf("committer: %s\n", msg.Commiter)
	fmt.Printf("number: %d\nround: %d\n", msg.BlockHeight, msg.Round)
	fmt.Printf("hash: %s\n", msg.BlockHash)

}

func CopyCmtMsg(msg *CommitMsg) *CommitMsg {
	cpy := *msg

	cpy.BlockHeight = msg.BlockHeight

	if len(msg.CommitSig) > 0 {
		cpy.CommitSig = make([]byte, len(msg.CommitSig))
		copy(cpy.CommitSig, msg.CommitSig)
	}

	return &cpy
}
