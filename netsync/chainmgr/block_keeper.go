package chainmgr

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/consensus"
	"github.com/vapor/errors"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/p2p/security"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	syncCycle = 5 * time.Second

	noNeedSync = iota
	fastSyncType
	regularSyncType
)

var (
	syncTimeout = 30 * time.Second

	errRequestTimeout = errors.New("request timeout")
	errPeerDropped    = errors.New("Peer dropped")
)

type FastSync interface {
	locateBlocks(locator []*bc.Hash, stopHash *bc.Hash) ([]*types.Block, error)
	locateHeaders(locator []*bc.Hash, stopHash *bc.Hash, skip uint64, maxNum uint64) ([]*types.BlockHeader, error)
	process() error
	setSyncPeer(peer *peers.Peer)
}

type Fetcher interface {
	processBlock(peerID string, block *types.Block)
	processBlocks(peerID string, blocks []*types.Block)
	processHeaders(peerID string, headers []*types.BlockHeader)
	requireBlock(peerID string, height uint64) (*types.Block, error)
}

type blockMsg struct {
	block  *types.Block
	peerID string
}

type blocksMsg struct {
	blocks []*types.Block
	peerID string
}

type headersMsg struct {
	headers []*types.BlockHeader
	peerID  string
}

type blockKeeper struct {
	chain      Chain
	fastSync   FastSync
	msgFetcher Fetcher
	peers      *peers.PeerSet
	syncPeer   *peers.Peer

	quit chan struct{}
}

func newBlockKeeper(chain Chain, peers *peers.PeerSet) *blockKeeper {
	msgFetcher := newMsgFetcher(peers)
	return &blockKeeper{
		chain:      chain,
		fastSync:   newFastSync(chain, msgFetcher, peers),
		msgFetcher: msgFetcher,
		peers:      peers,
		quit:       make(chan struct{}),
	}
}

func (bk *blockKeeper) locateBlocks(locator []*bc.Hash, stopHash *bc.Hash) ([]*types.Block, error) {
	return bk.fastSync.locateBlocks(locator, stopHash)
}

func (bk *blockKeeper) locateHeaders(locator []*bc.Hash, stopHash *bc.Hash, skip uint64, maxNum uint64) ([]*types.BlockHeader, error) {
	return bk.fastSync.locateHeaders(locator, stopHash, skip, maxNum)
}

func (bk *blockKeeper) processBlock(peerID string, block *types.Block) {
	bk.msgFetcher.processBlock(peerID, block)
}

func (bk *blockKeeper) processBlocks(peerID string, blocks []*types.Block) {
	bk.msgFetcher.processBlocks(peerID, blocks)
}

func (bk *blockKeeper) processHeaders(peerID string, headers []*types.BlockHeader) {
	bk.msgFetcher.processHeaders(peerID, headers)
}

func (bk *blockKeeper) regularBlockSync() error {
	peerHeight := bk.syncPeer.Height()
	bestHeight := bk.chain.BestBlockHeight()
	i := bestHeight + 1
	for i <= peerHeight {
		block, err := bk.msgFetcher.requireBlock(bk.syncPeer.ID(), i)
		if err != nil {
			bk.peers.ErrorHandler(bk.syncPeer.ID(), security.LevelConnException, err)
			return err
		}

		isOrphan, err := bk.chain.ProcessBlock(block)
		if err != nil {
			bk.peers.ErrorHandler(bk.syncPeer.ID(), security.LevelMsgIllegal, err)
			return err
		}

		if isOrphan {
			i--
			continue
		}
		i = bk.chain.BestBlockHeight() + 1
	}
	log.WithFields(log.Fields{"module": logModule, "height": bk.chain.BestBlockHeight()}).Info("regular sync success")
	return nil
}

func (bk *blockKeeper) start() {
	go bk.syncWorker()
}

func (bk *blockKeeper) checkSyncType() int {
	peer := bk.peers.BestIrreversiblePeer(consensus.SFFullNode | consensus.SFFastSync)
	if peer == nil {
		log.WithFields(log.Fields{"module": logModule}).Debug("can't find fast sync peer")
		return noNeedSync
	}

	bestHeight := bk.chain.BestBlockHeight()

	if peerIrreversibleHeight := peer.IrreversibleHeight(); peerIrreversibleHeight >= bestHeight+minGapStartFastSync {
		bk.fastSync.setSyncPeer(peer)
		return fastSyncType
	}

	peer = bk.peers.BestPeer(consensus.SFFullNode)
	if peer == nil {
		log.WithFields(log.Fields{"module": logModule}).Debug("can't find sync peer")
		return noNeedSync
	}

	peerHeight := peer.Height()
	if peerHeight > bestHeight {
		bk.syncPeer = peer
		return regularSyncType
	}

	return noNeedSync
}

func (bk *blockKeeper) startSync() bool {
	switch bk.checkSyncType() {
	case fastSyncType:
		if err := bk.fastSync.process(); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("failed on fast sync")
			return false
		}
	case regularSyncType:
		if err := bk.regularBlockSync(); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on regularBlockSync")
			return false
		}
	}

	return true
}

func (bk *blockKeeper) stop() {
	close(bk.quit)
}

func (bk *blockKeeper) syncWorker() {
	syncTicker := time.NewTicker(syncCycle)
	defer syncTicker.Stop()

	for {
		select {
		case <-syncTicker.C:
			if update := bk.startSync(); !update {
				continue
			}

			if err := bk.peers.BroadcastNewStatus(bk.chain.BestBlockHeader(), bk.chain.BestIrreversibleHeader()); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on syncWorker broadcast new status")
			}
		case <-bk.quit:
			return
		}
	}
}
