package bbft

import (
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/consensus"
)

type consensusManger struct {
	peers *peerSet
}

//BasePeer is the interface for connection level peer
type BasePeer interface {
	Addr() net.Addr
	ID() string
	ServiceFlag() consensus.ServiceFlag
	TrySend(byte, interface{}) bool
	IsLAN() bool
}

func (cm *consensusManger) processMsg(basePeer BasePeer, msgType byte, msg BlockchainMessage) {
	peer := cm.peers.getPeer(basePeer.ID())
	if peer == nil {
		return
	}

	log.WithFields(log.Fields{
		"module":  logModule,
		"peer":    basePeer.Addr(),
		"type":    reflect.TypeOf(msg),
		"message": msg.String(),
	}).Info("receive message from peer")

	switch msg := msg.(type) {
	case *BlockProposeMessage:
		cm.handleBlockProposeMsg(peer, msg)

	case *BlockSigMessage:
		cm.handleBlockSigMsg(peer, msg)

	default:
		log.WithFields(log.Fields{
			"module":       logModule,
			"peer":         basePeer.Addr(),
			"message_type": reflect.TypeOf(msg),
		}).Error("unhandled message type")
	}
}

func (cm *consensusManger) handleBlockProposeMsg(peer *peer, msg *BlockProposeMessage) {
}

func (cm *consensusManger) handleBlockSigMsg(peer *peer, msg *BlockSigMessage) {
}
