package bbft

import (
	"bytes"
	"errors"

	"github.com/tendermint/go-wire"
)

//Consensus msg byte
const (
	ConsensusChannel = byte(0x50)

	BlockSigByte     = byte(0x10)
	BlockProposeByte = byte(0x11)

	maxBlockchainResponseSize = 22020096 + 2
)

//BlockchainMessage is a generic message for this reactor.
type ConsensusMessage interface {
	String() string
}

var _ = wire.RegisterInterface(
	struct{ ConsensusMessage }{},
	wire.ConcreteType{&BlockSigMessage{}, BlockSigByte},
	wire.ConcreteType{&BlockProposeMessage{}, BlockProposeByte},
)

//DecodeMessage decode msg
func DecodeMessage(bz []byte) (msgType byte, msg ConsensusMessage, err error) {
	msgType = bz[0]
	n := int(0)
	r := bytes.NewReader(bz)
	msg = wire.ReadBinary(struct{ ConsensusMessage }{}, r, maxBlockchainResponseSize, &n, &err).(struct{ ConsensusMessage }).ConsensusMessage
	if err != nil && n != len(bz) {
		err = errors.New("DecodeMessage() had bytes left over")
	}
	return
}

type BlockSigMessage struct {
}

func (bs *BlockSigMessage) String() string {
	return ""
}

type BlockProposeMessage struct {
}

func (bp *BlockProposeMessage) String() string {
	return ""
}
