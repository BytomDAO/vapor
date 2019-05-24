package consensus

import (
	"reflect"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/event"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/p2p"
	"github.com/vapor/protocol/bc/types"
)

type Manager struct {
	sw           Switch
	peers        *peers.PeerSet
	blockFetcher *blockFetcher
	chain        Chain

	quit chan struct{}

	eventDispatcher  *event.Dispatcher
	proposedBlockSub *event.Subscription
	sendBlockSignSub *event.Subscription
}

type Switch interface {
	AddReactor(name string, reactor p2p.Reactor) p2p.Reactor
	AddBannedPeer(string) error
}

// Chain is the interface for Bytom core
type Chain interface {
	BestBlockHeight() uint64
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
		quit:            make(chan struct{}),
		eventDispatcher: dispatcher,
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

	log.WithFields(log.Fields{"module": logModule, "peer": peerID, "type": reflect.TypeOf(msg), "message": msg.String()}).Info("receive message from peer")

	switch msg := msg.(type) {
	case *BlockProposeMessage:
		m.handleBlockProposeMsg(peerID, msg)

	case *BlockSignMessage:
		m.handleBlockSigMsg(peerID, msg)

	default:
		log.WithFields(log.Fields{"module": logModule, "peer": peerID, "message_type": reflect.TypeOf(msg)}).Error("unhandled message type")
	}
}

func (m *Manager) handleBlockProposeMsg(peerID string, msg *BlockProposeMessage) {
	block, err := msg.GetProposeBlock()
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleMineBlockMsg GetMineBlock")
		return
	}

	hash := block.Hash()
	m.peers.MarkBlock(peerID, &hash)
	m.blockFetcher.processNewBlock(&blockMsg{peerID: peerID, block: block})
}

func (m *Manager) handleBlockSigMsg(peerID string, msg *BlockSignMessage) {
	if err := m.eventDispatcher.Post(event.BlockSignEvent{PeerID: []byte(peerID), BlockID: msg.BlockID, Height: msg.Height, Sign: msg.Sign, Pubkey: msg.Pubkey}); err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("failed post block sign event")
	}
}

func (m *Manager) proposedBlockBroadcastLoop() {
	for {
		select {
		case obj, ok := <-m.proposedBlockSub.Chan():
			if !ok {
				log.WithFields(log.Fields{"module": logModule}).Warning("mined block subscription channel closed")
				return
			}

			ev, ok := obj.Data.(event.NewProposedBlockEvent)
			if !ok {
				log.WithFields(log.Fields{"module": logModule}).Error("event type error")
				continue
			}

			proposedMsg, err := NewBlockProposeBroadcastMsg(&ev.Block, ConsensusChannel)
			if err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("blockFetcher fail on create new propose block msg")
				return
			}
			if err := m.peers.BroadcastMsg(proposedMsg); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on broadcast mine block")
				continue
			}

		case <-m.quit:
			return
		}
	}
}

func (m *Manager) blockSignBroadcastLoop() {
	for {
		select {
		case obj, ok := <-m.sendBlockSignSub.Chan():
			if !ok {
				log.WithFields(log.Fields{"module": logModule}).Warning("send block sign subscription channel closed")
				return
			}

			ev, ok := obj.Data.(event.SendBlockSignEvent)
			if !ok {
				log.WithFields(log.Fields{"module": logModule}).Error("event type error")
				continue
			}

			blockSignMsg := NewBlockSignBroadcastMsg(ev.BlockID, ev.Height, ev.Sign, ev.Pubkey, ConsensusChannel)
			if err := m.peers.BroadcastMsg(blockSignMsg); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("failed to broadcast block sign message.")
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
	m.proposedBlockSub, err = m.eventDispatcher.Subscribe(event.NewProposedBlockEvent{})
	if err != nil {
		return err
	}

	m.sendBlockSignSub, err = m.eventDispatcher.Subscribe(event.SendBlockSignEvent{})
	if err != nil {
		return err
	}

	go m.proposedBlockBroadcastLoop()
	go m.blockSignBroadcastLoop()
	return nil
}

//Stop stop sync manager
func (m *Manager) Stop() {
	close(m.quit)
	m.proposedBlockSub.Unsubscribe()
	m.sendBlockSignSub.Unsubscribe()
}
