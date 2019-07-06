package chainmgr

import (
	"errors"
	"sync"
)

var ErrNoValidFastSyncPeer = errors.New("no valid fast sync peer")

type fastSyncPeers struct {
	peers map[string]bool
	mtx   sync.RWMutex
}

func newFastSyncPeers() *fastSyncPeers {
	return &fastSyncPeers{
		peers: make(map[string]bool),
	}
}

func (fs *fastSyncPeers) add(peerID string) {
	fs.mtx.RLock()
	defer fs.mtx.RUnlock()

	fs.peers[peerID] = false
}

func (fs *fastSyncPeers) selectIdlePeers() []string {
	fs.mtx.RLock()
	defer fs.mtx.RUnlock()

	peers := make([]string, 0)
	for peerID, isBusy := range fs.peers {
		if isBusy {
			continue
		}

		fs.peers[peerID] = true
		peers = append(peers, peerID)
	}

	return peers
}

func (fs *fastSyncPeers) selectIdlePeer() (string, error) {
	fs.mtx.RLock()
	defer fs.mtx.RUnlock()

	for peerID, isBusy := range fs.peers {
		if isBusy {
			continue
		}

		fs.peers[peerID] = true
		return peerID, nil
	}

	return "", ErrNoValidFastSyncPeer
}

func (fs *fastSyncPeers) setIdle(peerID string) {
	fs.mtx.RLock()
	defer fs.mtx.RUnlock()

	_, ok := fs.peers[peerID]
	if !ok {
		return
	}

	fs.peers[peerID] = false
}

func (fs *fastSyncPeers) size() int {
	fs.mtx.RLock()
	defer fs.mtx.RUnlock()

	return len(fs.peers)
}
