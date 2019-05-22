package netsync

import (
	"github.com/vapor/netsync/peers"
	"github.com/vapor/protocol/bc/types"
)

//type BroadcastProcess interface {
//	filter(ps *PeerSet) []*Peer
//	mark(*Peer)
//	getChan() byte
//	getMsg() interface{}
//	msgString() string
//}

type minedBlockBroadcastMsg struct {
	block     *types.Block
	msg       *MineBlockMessage
	transChan byte
}

func newMinedBlockBroadcastMsg(block *types.Block, transChan byte) (*minedBlockBroadcastMsg, error) {
	msg, err := NewMinedBlockMessage(block)
	if err != nil {
		return nil, err
	}
	return &minedBlockBroadcastMsg{block: block, msg: msg, transChan: transChan}, nil
}

func (m *minedBlockBroadcastMsg) GetChan() byte {
	return m.transChan
}

func (m *minedBlockBroadcastMsg) GetMsg() interface{} {
	return m.msg
}

func (m *minedBlockBroadcastMsg) MsgString() string {
	return m.msg.String()
}

func (m *minedBlockBroadcastMsg) Mark(ps *peers.PeerSet, peers []string) {
	hash := m.block.Hash()
	height := m.block.Height
	for _, peer := range peers {
		ps.MarkBlock(peer, &hash)
		ps.MarkStatus(peer, height)
	}
}

func (m *minedBlockBroadcastMsg) Filter(ps *peers.PeerSet) []string {
	//TODO: SPV NODE FILTER
	return ps.PeersWithoutBlock(m.block.Hash())
}

type statusBroadcastMsg struct {
	header    *types.BlockHeader
	msg       *StatusMessage
	transChan byte
}

func newStatusBroadcastMsg(header *types.BlockHeader, transChan byte) (*statusBroadcastMsg, error) {
	msg := NewStatusMessage(header)
	return &statusBroadcastMsg{header: header, msg: msg, transChan: transChan}, nil
}

func (s *statusBroadcastMsg) GetChan() byte {
	return s.transChan
}

func (s *statusBroadcastMsg) GetMsg() interface{} {
	return s.msg
}

func (s *statusBroadcastMsg) MsgString() string {
	return s.msg.String()
}

func (s *statusBroadcastMsg) Filter(ps *peers.PeerSet) []string {
	return ps.PeersWithoutNewStatus(s.header.Height)
}

func (s *statusBroadcastMsg) Mark(ps *peers.PeerSet, peers []string) {
	height := s.header.Height
	for _, peer := range peers {
		ps.MarkStatus(peer, height)
	}
}

type txBroadcastMsg struct {
	tx        *types.Tx
	msg       *TransactionMessage
	transChan byte
}

func newTxBroadcastMsg(tx *types.Tx, transChan byte) (*txBroadcastMsg, error) {
	msg, _ := NewTransactionMessage(tx)
	return &txBroadcastMsg{tx: tx, msg: msg, transChan: transChan}, nil
}

func (t *txBroadcastMsg) GetChan() byte {
	return t.transChan
}

func (t *txBroadcastMsg) GetMsg() interface{} {
	return t.msg
}

func (t *txBroadcastMsg) MsgString() string {
	return t.msg.String()
}

func (t *txBroadcastMsg) Filter(ps *peers.PeerSet) []string {
	//TODO: 		if peer.isSPVNode() && !peer.isRelatedTx(tx) {
	return ps.PeersWithoutTx(t.tx.ID)
}

func (t *txBroadcastMsg) Mark(ps *peers.PeerSet, peers []string) {
	for _, peer := range peers {
		ps.MarkTx(peer, &t.tx.ID)
	}
}
