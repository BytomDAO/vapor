package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/vapor/common"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto"
	"github.com/vapor/encoding/blockchain"
	"github.com/vapor/encoding/bufpool"
	"github.com/vapor/errors"
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

func (msg *PreprepareMsg) MarshalText() ([]byte, error) {
	buf := bufpool.Get()
	defer bufpool.Put(buf)

	if _, err := blockchain.WriteVarint31(buf, uint64(msg.Round)); err != nil {
		return nil, err
	}

	block, err := msg.Block.MarshalText()
	if err != nil {
		return nil, err
	}
	if _, err := blockchain.WriteVarstr31(buf, block); err != nil {
		return nil, err
	}

	enc := make([]byte, hex.EncodedLen(buf.Len()))
	hex.Encode(enc, buf.Bytes())
	return enc, nil
}

// UnmarshalText fulfills the encoding.TextUnmarshaler interface.
func (msg *PreprepareMsg) UnmarshalText(text []byte) error {
	decoded := make([]byte, hex.DecodedLen(len(text)))
	if _, err := hex.Decode(decoded, text); err != nil {
		return err
	}

	r := blockchain.NewReader(decoded)

	round, err := blockchain.ReadVarint31(r)
	if err != nil {
		return errors.Wrap(err, "reading  round of bftpreprepare")
	}
	msg.Round = round

	block, err := blockchain.ReadVarstr31(r)
	if err != nil {
		return errors.Wrap(err, "reading  block of bftpreprepare")
	}

	if err := msg.Block.UnmarshalText(block); err != nil {
		return err
	}

	if trailing := r.Len(); trailing > 0 {
		return fmt.Errorf("trailing garbage (%d bytes)", trailing)
	}
	return nil
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

func (msg *PrepareMsg) MarshalText() ([]byte, error) {
	buf := bufpool.Get()
	defer bufpool.Put(buf)

	if _, err := blockchain.WriteVarint31(buf, uint64(msg.Round)); err != nil {
		return nil, err
	}

	address, err := common.DecodeAddress(msg.PrepareAddr, &consensus.ActiveNetParams)
	if err != nil {
		return nil, err
	}
	redeemContract := address.ScriptAddress()

	if _, err := blockchain.WriteVarstr31(buf, redeemContract); err != nil {
		return nil, err
	}

	if _, err := blockchain.WriteVarint63(buf, msg.BlockHeight); err != nil {
		return nil, err
	}

	blockHash, err := msg.BlockHash.MarshalText()
	if err != nil {
		return nil, err
	}

	if _, err := blockchain.WriteVarstr31(buf, blockHash); err != nil {
		return nil, err
	}

	if _, err := blockchain.WriteVarstr31(buf, msg.PrepareSig); err != nil {
		return nil, err
	}

	enc := make([]byte, hex.EncodedLen(buf.Len()))
	hex.Encode(enc, buf.Bytes())
	return enc, nil
}

// UnmarshalText fulfills the encoding.TextUnmarshaler interface.
func (msg *PrepareMsg) UnmarshalText(text []byte) error {
	decoded := make([]byte, hex.DecodedLen(len(text)))
	if _, err := hex.Decode(decoded, text); err != nil {
		return err
	}

	r := blockchain.NewReader(decoded)

	round, err := blockchain.ReadVarint31(r)
	if err != nil {
		return errors.Wrap(err, "reading  round of bftpreprepare")
	}
	msg.Round = round

	redeemContract, err := blockchain.ReadVarstr31(r)
	if err != nil {
		return errors.Wrap(err, "reading  address of bftpreprepare")
	}
	address, err := common.NewPeginAddressWitnessScriptHash(redeemContract, &consensus.ActiveNetParams)
	if err != nil {
		return errors.Wrap(err, "reading  address of bftpreprepare")
	}

	msg.PrepareAddr = address.EncodeAddress()

	hight, err := blockchain.ReadVarint63(r)
	if err != nil {
		return errors.Wrap(err, "reading  hight of bftpreprepare")
	}
	msg.BlockHeight = hight

	blockHash, err := blockchain.ReadVarstr31(r)

	if err := msg.BlockHash.UnmarshalText(blockHash); err != nil {
		return err
	}

	if msg.PrepareSig, err = blockchain.ReadVarstr31(r); err != nil {
		return err
	}

	if trailing := r.Len(); trailing > 0 {
		return fmt.Errorf("trailing garbage (%d bytes)", trailing)
	}
	return nil
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
	fmt.Printf("hash: %s\n", msg.BlockHash.String())

}

func (msg *CommitMsg) MarshalText() ([]byte, error) {
	buf := bufpool.Get()
	defer bufpool.Put(buf)

	if _, err := blockchain.WriteVarint31(buf, uint64(msg.Round)); err != nil {
		return nil, err
	}

	address, err := common.DecodeAddress(msg.Commiter, &consensus.ActiveNetParams)
	if err != nil {
		return nil, err
	}
	redeemContract := address.ScriptAddress()

	if _, err := blockchain.WriteVarstr31(buf, redeemContract); err != nil {
		return nil, err
	}

	if _, err := blockchain.WriteVarint63(buf, msg.BlockHeight); err != nil {
		return nil, err
	}

	blockHash, err := msg.BlockHash.MarshalText()
	if err != nil {
		return nil, err
	}

	if _, err := blockchain.WriteVarstr31(buf, blockHash); err != nil {
		return nil, err
	}

	if _, err := blockchain.WriteVarstr31(buf, msg.CommitSig); err != nil {
		return nil, err
	}

	enc := make([]byte, hex.EncodedLen(buf.Len()))
	hex.Encode(enc, buf.Bytes())
	return enc, nil
}

// UnmarshalText fulfills the encoding.TextUnmarshaler interface.
func (msg *CommitMsg) UnmarshalText(text []byte) error {
	decoded := make([]byte, hex.DecodedLen(len(text)))
	if _, err := hex.Decode(decoded, text); err != nil {
		return err
	}

	r := blockchain.NewReader(decoded)

	round, err := blockchain.ReadVarint31(r)
	if err != nil {
		return errors.Wrap(err, "reading  round of bftpreprepare")
	}
	msg.Round = round

	redeemContract, err := blockchain.ReadVarstr31(r)
	if err != nil {
		return errors.Wrap(err, "reading  address of bftpreprepare")
	}
	address, err := common.NewPeginAddressWitnessScriptHash(redeemContract, &consensus.ActiveNetParams)
	if err != nil {
		return errors.Wrap(err, "reading  address of bftpreprepare")
	}

	msg.Commiter = address.EncodeAddress()

	hight, err := blockchain.ReadVarint63(r)
	if err != nil {
		return errors.Wrap(err, "reading  hight of bftpreprepare")
	}
	msg.BlockHeight = hight

	blockHash, err := blockchain.ReadVarstr31(r)

	if err := msg.BlockHash.UnmarshalText(blockHash); err != nil {
		return err
	}

	if msg.CommitSig, err = blockchain.ReadVarstr31(r); err != nil {
		return err
	}

	if trailing := r.Len(); trailing > 0 {
		return fmt.Errorf("trailing garbage (%d bytes)", trailing)
	}
	return nil
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
