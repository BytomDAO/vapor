package peers

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/consensus"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
)

/*
//添加删除查询节点
AddPeer(peer BasePeer)
RemovePeer(peerID string)
GetPeer(id string) *peer
BestPeerInfo(flag consensus.ServiceFlag) (string, uint64)
GetPeerInfo(peerID string) *PeerInfo
GetPeerInfos() []*PeerInfo
SetStatus(peerID string, height uint64, hash *bc.Hash)

//节点错误处理
AddBanScore(peerID string, persistent, transient uint32, reason string)
ErrorHandler(peerID string, err error)
*/

//BasePeerSet is the intergace for connection level peer manager
type BasePeerSet interface {
	AddBannedPeer(string) error
	StopPeerGracefully(string)
}

type PeerSet struct {
	BasePeerSet
	mtx   sync.RWMutex
	peers map[string]*Peer
}

// NewPeerSet creates a new Peer set to track the active participants.
func NewPeerSet(basePeerSet BasePeerSet) *PeerSet {
	return &PeerSet{
		BasePeerSet: basePeerSet,
		peers:       make(map[string]*Peer),
	}
}

func (ps *PeerSet) AddBanScore(peerID string, persistent, transient uint32, reason string) {
	ps.mtx.Lock()
	peer := ps.peers[peerID]
	ps.mtx.Unlock()

	if peer == nil {
		return
	}
	if ban := peer.addBanScore(persistent, transient, reason); !ban {
		return
	}
	if err := ps.AddBannedPeer(peer.Addr().String()); err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on add ban peer")
	}
	ps.RemovePeer(peerID)
}

func (ps *PeerSet) AddPeer(peer BasePeer) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if _, ok := ps.peers[peer.ID()]; !ok {
		ps.peers[peer.ID()] = newPeer(peer)
		return
	}
	log.WithField("module", logModule).Warning("add existing peer to blockKeeper")
}

func (ps *PeerSet) BestPeerInfo(flag consensus.ServiceFlag) (string, uint64) {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	var bestPeer *Peer
	for _, p := range ps.peers {
		if !p.services.IsEnable(flag) {
			continue
		}
		if bestPeer == nil || p.height > bestPeer.height || (p.height == bestPeer.height && p.IsLAN()) {
			bestPeer = p
		}
	}
	if bestPeer == nil {
		return "", 0
	}
	return bestPeer.ID(), bestPeer.bestHeight()
}

func (ps *PeerSet) ErrorHandler(peerID string, err error) {
	if errors.Root(err) == ErrPeerMisbehave {
		ps.AddBanScore(peerID, 20, 0, err.Error())
	} else {
		ps.RemovePeer(peerID)
	}
}

// Peer retrieves the registered peer with the given id.
func (ps *PeerSet) GetPeer(id string) *Peer {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()
	return ps.peers[id]
}

func (ps *PeerSet) GetPeerInfo(peerID string) *PeerInfo {
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return nil
	}

	return peer.getPeerInfo()
}

func (ps *PeerSet) GetPeerInfos() []*PeerInfo {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	result := []*PeerInfo{}
	for _, peer := range ps.peers {
		result = append(result, peer.getPeerInfo())
	}
	return result
}

func (ps *PeerSet) RemovePeer(peerID string) {
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return
	}

	ps.mtx.Lock()
	delete(ps.peers, peerID)
	ps.mtx.Unlock()
	ps.StopPeerGracefully(peerID)
}

func (ps *PeerSet) SetStatus(peerID string, height uint64, hash *bc.Hash) {
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return
	}

	peer.setStatus(height, hash)
}
