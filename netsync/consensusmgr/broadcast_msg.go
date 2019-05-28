package consensusmgr

import (
	"github.com/vapor/netsync/peers"
)

type BroadcastMsg struct {
	msg ConsensusMessage

	transChan byte
}

func NewBroadcastMsg(msg ConsensusMessage, transChan byte) *BroadcastMsg {
	return &BroadcastMsg{
		msg:       msg,
		transChan: transChan,
	}
}

func (b *BroadcastMsg) GetChan() byte {
	return b.transChan
}

func (b *BroadcastMsg) GetMsg() interface{} {
	return struct{ ConsensusMessage }{b.msg}
}

func (b *BroadcastMsg) MsgString() string {
	return b.msg.String()
}

func (b *BroadcastMsg) MarkSendRecord(ps *peers.PeerSet, peers []string) {
	b.msg.BroadcastMarkSendRecord(ps, peers)
}

func (b *BroadcastMsg) FilterTargetPeers(ps *peers.PeerSet) []string {
	return b.msg.BroadcastFilterTargetPeers(ps)
}
