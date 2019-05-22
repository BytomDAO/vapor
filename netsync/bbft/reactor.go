package bbft

import (
	log "github.com/sirupsen/logrus"

	"github.com/vapor/p2p"
	"github.com/vapor/p2p/connection"
)

const logModule = "bbft"

type ConsensusReactor struct {
	p2p.BaseReactor
	c     *consensus
	peers *peerSet
}

func NewConsensusReactor(peers *peerSet) *ConsensusReactor {
	cr := &ConsensusReactor{
		peers: peers,
	}
	cr.BaseReactor = *p2p.NewBaseReactor("ConsensusReactor", cr)
	return cr
}

// GetChannels implements Reactor
func (cr *ConsensusReactor) GetChannels() []*connection.ChannelDescriptor {
	return []*connection.ChannelDescriptor{
		{
			ID:                ConsensusChannel,
			Priority:          10,
			SendQueueCapacity: 100,
		},
	}
}

// OnStart implements BaseService
func (cr *ConsensusReactor) OnStart() error {
	cr.BaseReactor.OnStart()
	return nil
}

// OnStop implements BaseService
func (cr *ConsensusReactor) OnStop() {
	cr.BaseReactor.OnStop()
}

// AddPeer implements Reactor by sending our state to peer.
func (cr *ConsensusReactor) AddPeer(peer *p2p.Peer) error {
	return nil
}

// RemovePeer implements Reactor by removing peer from the pool.
func (cr *ConsensusReactor) RemovePeer(peer *p2p.Peer, reason interface{}) {
}

// Receive implements Reactor by handling 4 types of messages (look below).
func (cr *ConsensusReactor) Receive(chID byte, src *p2p.Peer, msgBytes []byte) {
	msgType, msg, err := DecodeMessage(msgBytes)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on reactor decoding message")
		return
	}

	cr.c.processMsg(src, msgType, msg)
}
