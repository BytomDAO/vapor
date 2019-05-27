package consensusmgr

import (
	"reflect"

	"github.com/sirupsen/logrus"

	"github.com/vapor/event"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/p2p"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

type Manager struct {
	sw           Switch
	chain        Chain
	peers        *peers.PeerSet
	blockFetcher *blockFetcher

	eventDispatcher      *event.Dispatcher
	blockProposeMsgSub   *event.Subscription
	BlockSignatureMsgSub *event.Subscription

	quit chan struct{}
}

type Switch interface {
	AddReactor(name string, reactor p2p.Reactor) p2p.Reactor
	AddBannedPeer(string) error
	ID() [32]byte
}

// Chain is the interface for Bytom core
type Chain interface {
	BestBlockHeight() uint64
	GetHeaderByHash(*bc.Hash) (*types.BlockHeader, error)
	ProcessBlock(*types.Block) (bool, error)
}

type blockMsg struct {
	block  *types.Block
	peerID string
}

func NewManager(sw Switch, chain Chain, dispatcher *event.Dispatcher, peers *peers.PeerSet) *Manager {
	manager := &Manager{
		sw:              sw,
		peers:           peers,
		blockFetcher:    newBlockFetcher(chain, peers),
		eventDispatcher: dispatcher,
		quit:            make(chan struct{}),
	}
	protocolReactor := NewConsensusReactor(manager)
	manager.sw.AddReactor("CONSENSUS", protocolReactor)
	return manager
}

func (m *Manager) AddPeer(peer peers.BasePeer) {
	m.peers.AddPeer(peer)
}

func (m *Manager) processMsg(peerID string, msgType byte, msg ConsensusMessage) {
	peer := m.peers.GetPeer(peerID)
	if peer == nil {
		return
	}

	logrus.WithFields(logrus.Fields{"module": logModule, "peer": peerID, "type": reflect.TypeOf(msg), "message": msg.String()}).Info("receive message from peer")

	switch msg := msg.(type) {
	case *BlockProposeMsg:
		m.handleBlockProposeMsg(peerID, msg)

	case *BlockSignatureMsg:
		m.handleBlockSignatureMsg(peerID, msg)

	default:
		logrus.WithFields(logrus.Fields{"module": logModule, "peer": peerID, "message_type": reflect.TypeOf(msg)}).Error("unhandled message type")
	}
}

func (m *Manager) handleBlockProposeMsg(peerID string, msg *BlockProposeMsg) {
	block, err := msg.GetProposeBlock()
	if err != nil {
		logrus.WithFields(logrus.Fields{"module": logModule, "err": err}).Warning("failed on get propose block")
		return
	}

	hash := block.Hash()
	m.peers.MarkBlock(peerID, &hash)
	m.blockFetcher.processNewBlock(&blockMsg{peerID: peerID, block: block})
}

func (m *Manager) handleBlockSignatureMsg(peerID string, msg *BlockSignatureMsg) {
	var id [32]byte
	copy(id[:], peerID)
	if err := m.eventDispatcher.Post(event.ReceivedBlockSignatureEvent{BlockID: msg.BlockID, Height: msg.Height, Signature: msg.Signature, Pubkey: id}); err != nil {
		logrus.WithFields(logrus.Fields{"module": logModule, "err": err}).Error("failed on post block signature event")
	}
}

func (m *Manager) blockProposeMsgBroadcastLoop() {
	for {
		select {
		case obj, ok := <-m.blockProposeMsgSub.Chan():
			if !ok {
				logrus.WithFields(logrus.Fields{"module": logModule}).Warning("blockProposeMsgSub channel closed")
				return
			}

			ev, ok := obj.Data.(event.NewBlockProposeEvent)
			if !ok {
				logrus.WithFields(logrus.Fields{"module": logModule}).Error("event type error")
				continue
			}

			proposeMsg, err := NewBlockProposeBroadcastMsg(&ev.Block, ConsensusChannel)
			if err != nil {
				logrus.WithFields(logrus.Fields{"module": logModule, "err": err}).Error("failed on create BlockProposeBroadcastMsg")
				return
			}
			if err := m.peers.BroadcastMsg(proposeMsg); err != nil {
				logrus.WithFields(logrus.Fields{"module": logModule, "err": err}).Error("failed on broadcast BlockProposeBroadcastMsg")
				continue
			}

		case <-m.quit:
			return
		}
	}
}

func (m *Manager) blockSignatureMsgBroadcastLoop() {
	for {
		select {
		case obj, ok := <-m.blockProposeMsgSub.Chan():
			if !ok {
				logrus.WithFields(logrus.Fields{"module": logModule}).Warning("blockProposeMsgSub channel closed")
				return
			}

			ev, ok := obj.Data.(event.BlockSignatureEvent)
			if !ok {
				logrus.WithFields(logrus.Fields{"module": logModule}).Error("event type error")
				continue
			}
			blockHeader, err := m.chain.GetHeaderByHash(&ev.BlockHash)
			if err != nil {
				logrus.WithFields(logrus.Fields{"module": logModule, "err": err}).Error("failed on get header by hash from chain.")
				return
			}
			blockSignatureMsg := NewSignatureBroadcastMsg(ev.BlockHash.Byte32(), blockHeader.Height, ev.Signature, m.sw.ID(), ConsensusChannel)
			if err := m.peers.BroadcastMsg(blockSignatureMsg); err != nil {
				logrus.WithFields(logrus.Fields{"module": logModule, "err": err}).Error("failed on broadcast BlockSignBroadcastMsg.")
				return
			}

		case <-m.quit:
			return
		}
	}
}

func (m *Manager) RemovePeer(peerID string) {
	m.peers.RemovePeer(peerID)
}

func (m *Manager) Start() error {
	var err error
	m.blockProposeMsgSub, err = m.eventDispatcher.Subscribe(event.NewBlockProposeEvent{})
	if err != nil {
		return err
	}

	m.BlockSignatureMsgSub, err = m.eventDispatcher.Subscribe(event.BlockSignatureEvent{})
	if err != nil {
		return err
	}

	go m.blockProposeMsgBroadcastLoop()
	go m.blockSignatureMsgBroadcastLoop()
	return nil
}

//Stop consensus manager
func (m *Manager) Stop() {
	close(m.quit)
	m.blockProposeMsgSub.Unsubscribe()
	m.BlockSignatureMsgSub.Unsubscribe()
}
