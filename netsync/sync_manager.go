package netsync

import (
	"errors"

	log "github.com/sirupsen/logrus"

	cfg "github.com/vapor/config"
	"github.com/vapor/consensus"
	"github.com/vapor/event"
	bbft "github.com/vapor/netsync/consensus"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/p2p"
	core "github.com/vapor/protocol"
)

type ChainMgr interface {
	Start() error
	IsCaughtUp() bool
	Stop()
}

type ConsensusMgr interface {
	Start() error
	Stop()
}

//SyncManager Sync Manager is responsible for the business layer information synchronization
type SyncManager struct {
	config       *cfg.Config
	sw           Switch
	chainMgr     ChainMgr
	consensusMgr ConsensusMgr
	peers        *peers.PeerSet
}

// CreateSyncManager create sync manager and set switch.
func NewSyncManager(config *cfg.Config, chain Chain, txPool *core.TxPool, dispatcher *event.Dispatcher) (*SyncManager, error) {
	sw, err := p2p.NewSwitch(config)
	if err != nil {
		return nil, err
	}
	peers := peers.NewPeerSet(sw)

	chainManger, err := NewChainManager(config, sw, chain, txPool, dispatcher, peers)
	if err != nil {
		return nil, err
	}
	consensusMgr := bbft.NewManager(sw, chain, dispatcher, peers)
	return &SyncManager{
		config:       config,
		sw:           sw,
		chainMgr:     chainManger,
		consensusMgr: consensusMgr,
		peers:        peers,
	}, nil
}

func (sm *SyncManager) Start() error {
	if _, err := sm.sw.Start(); err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("failed start switch")
		return err
	}

	if err := sm.chainMgr.Start(); err != nil {
		return err
	}

	return sm.consensusMgr.Start()
}

func (sm *SyncManager) Stop() {
	sm.chainMgr.Stop()
	sm.consensusMgr.Stop()
	if !sm.config.VaultMode {
		sm.sw.Stop()
	}

}

func (sm *SyncManager) IsListening() bool {
	if sm.config.VaultMode {
		return false
	}
	return sm.sw.IsListening()

}

//IsCaughtUp check wheather the peer finish the sync
func (sm *SyncManager) IsCaughtUp() bool {
	return sm.chainMgr.IsCaughtUp()
}

func (sm *SyncManager) PeerCount() int {
	if sm.config.VaultMode {
		return 0
	}
	return len(sm.sw.Peers().List())
}

func (sm *SyncManager) GetNetwork() string {
	return sm.config.ChainID
}

func (sm *SyncManager) BestPeer() *peers.PeerInfo {
	bestPeer := sm.peers.BestPeer(consensus.SFFullNode)
	if bestPeer != nil {
		return bestPeer.GetPeerInfo()
	}
	return nil
}

func (sm *SyncManager) DialPeerWithAddress(addr *p2p.NetAddress) error {
	if sm.config.VaultMode {
		return errVaultModeDialPeer
	}

	return sm.sw.DialPeerWithAddress(addr)
}

//GetPeerInfos return peer info of all peers
func (sm *SyncManager) GetPeerInfos() []*peers.PeerInfo {
	return sm.peers.GetPeerInfos()
}

//StopPeer try to stop peer by given ID
func (sm *SyncManager) StopPeer(peerID string) error {
	if peer := sm.peers.GetPeer(peerID); peer == nil {
		return errors.New("peerId not exist")
	}
	sm.peers.RemovePeer(peerID)
	return nil
}
