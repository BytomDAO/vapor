package chainmgr

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/consensus"
	"github.com/vapor/errors"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/p2p/security"
	"github.com/vapor/protocol/bc/types"
)

const (
	syncCycle            = 5 * time.Second
	blockProcessChSize   = 1024
	blocksProcessChSize  = 128
	headersProcessChSize = 1024
)

var (
	syncTimeout = 30 * time.Second

	errRequestTimeout = errors.New("request timeout")
	errPeerDropped    = errors.New("Peer dropped")
)

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
	chain Chain
	peers *peers.PeerSet

	syncPeer         *peers.Peer
	blockProcessCh   chan *blockMsg
	blocksProcessCh  chan *blocksMsg
	headersProcessCh chan *headersMsg

	skeleton       []*types.BlockHeader
	commonAncestor *types.BlockHeader
	fastSyncLength int

	quite chan struct{}
}

func newBlockKeeper(chain Chain, peers *peers.PeerSet) *blockKeeper {
	return &blockKeeper{
		chain:            chain,
		peers:            peers,
		blockProcessCh:   make(chan *blockMsg, blockProcessChSize),
		blocksProcessCh:  make(chan *blocksMsg, blocksProcessChSize),
		headersProcessCh: make(chan *headersMsg, headersProcessChSize),
		quite:            make(chan struct{}),
	}
}

func (bk *blockKeeper) processBlock(peerID string, block *types.Block) {
	bk.blockProcessCh <- &blockMsg{block: block, peerID: peerID}
}

func (bk *blockKeeper) processBlocks(peerID string, blocks []*types.Block) {
	bk.blocksProcessCh <- &blocksMsg{blocks: blocks, peerID: peerID}
}

func (bk *blockKeeper) processHeaders(peerID string, headers []*types.BlockHeader) {
	bk.headersProcessCh <- &headersMsg{headers: headers, peerID: peerID}
}

func (bk *blockKeeper) regularBlockSync(wantHeight uint64) error {
	i := bk.chain.BestBlockHeight() + 1
	for i <= wantHeight {
		block, err := bk.requireBlock(i)
		if err != nil {
			return err
		}

		isOrphan, err := bk.chain.ProcessBlock(block)
		if err != nil {
			return err
		}

		if isOrphan {
			i--
			continue
		}
		i = bk.chain.BestBlockHeight() + 1
	}
	return nil
}

func (bk *blockKeeper) requireBlock(height uint64) (*types.Block, error) {
	if ok := bk.syncPeer.GetBlockByHeight(height); !ok {
		return nil, errPeerDropped
	}

	timeout := time.NewTimer(syncTimeout)
	defer timeout.Stop()

	for {
		select {
		case msg := <-bk.blockProcessCh:
			if msg.peerID != bk.syncPeer.ID() {
				continue
			}
			if msg.block.Height != height {
				continue
			}
			return msg.block, nil
		case <-timeout.C:
			return nil, errors.Wrap(errRequestTimeout, "requireBlock")
		}
	}
}

func (bk *blockKeeper) start() {
	go bk.syncWorker()
}

func (bk *blockKeeper) startSync() bool {
	blockHeight := bk.chain.BestBlockHeight()
	peer := bk.peers.BestPeer(consensus.SFFastSync | consensus.SFFullNode)
	if peer != nil && peer.Height() >= blockHeight+uint64(minGapStartFastSync) {
		bk.syncPeer = peer
		if err := bk.fastSynchronize(); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on fastBlockSync")
			bk.peers.ErrorHandler(peer.ID(), security.LevelMsgIllegal, err)
			return false
		}
		return true
	}

	blockHeight = bk.chain.BestBlockHeight()
	peer = bk.peers.BestPeer(consensus.SFFullNode)
	if peer != nil && peer.Height() > blockHeight {
		bk.syncPeer = peer
		targetHeight := blockHeight + uint64(maxBlocksPerMsg)
		if targetHeight > peer.Height() {
			targetHeight = peer.Height()
		}

		if err := bk.regularBlockSync(targetHeight); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on regularBlockSync")
			bk.peers.ErrorHandler(peer.ID(), security.LevelMsgIllegal, err)
			return false
		}
		return true
	}
	return false
}

func (bk *blockKeeper) stop() {
	close(bk.quite)
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

			block, err := bk.chain.GetBlockByHeight(bk.chain.BestBlockHeight())
			if err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on syncWorker get best block")
			}

			if err = bk.peers.BroadcastNewStatus(block); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on syncWorker broadcast new status")
			}
		case <-bk.quite:
			return
		}
	}
}
