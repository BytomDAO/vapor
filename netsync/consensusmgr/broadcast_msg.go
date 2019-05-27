package consensusmgr

import (
	"github.com/vapor/netsync/peers"
	"github.com/vapor/protocol/bc/types"
)

type BroadcastMsg struct {
	signature []byte
	msg       *ConsensusMessage
	transChan byte
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
