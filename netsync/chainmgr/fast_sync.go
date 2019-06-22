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

var (
	maxBlocksPerMsg      = uint64(1000)
	maxHeadersPerMsg     = uint64(1000)
	fastSyncPivotGap     = uint64(64)
	minGapStartFastSync  = uint64(128)
	maxFastSyncBlocksNum = uint64(10000)

	errHeadersNum          = errors.New("headers number error")
	errExceedMaxHeadersNum = errors.New("exceed max headers number per msg")
	errBlocksNum           = errors.New("blocks number error")
	errNoCommonAncestor    = errors.New("can't find common ancestor")
	errNoSyncPeer          = errors.New("can't find sync peer")
)

type fastSync struct {
	chain          Chain
	peers          *peers.PeerSet
	syncPeer       *peers.Peer
	commonAncestor *types.BlockHeader
	stopHeader     *types.BlockHeader
	length         uint64

	blocksProcessCh  chan *blocksMsg
	headersProcessCh chan *headersMsg

	quite chan struct{}
}

func newFastSync(chain Chain, peers *peers.PeerSet) *fastSync {
	return &fastSync{
		chain:            chain,
		peers:            peers,
		blocksProcessCh:  make(chan *blocksMsg, blocksProcessChSize),
		headersProcessCh: make(chan *headersMsg, headersProcessChSize),
		quite:            make(chan struct{}),
	}
}

func (fs *fastSync) blockLocator() []*bc.Hash {
	header := fs.chain.BestBlockHeader()
	locator := []*bc.Hash{}

	step := uint64(1)
	for header != nil {
		headerHash := header.Hash()
		locator = append(locator, &headerHash)
		if header.Height == 0 {
			break
		}

		var err error
		if header.Height < step {
			header, err = fs.chain.GetHeaderByHeight(0)
		} else {
			header, err = fs.chain.GetHeaderByHeight(header.Height - step)
		}
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("blockKeeper fail on get blockLocator")
			break
		}

		if len(locator) >= 9 {
			step *= 2
		}
	}
	return locator
}

func (fs *fastSync) process() error {
	peer := fs.peers.BestPeer(consensus.SFFastSync | consensus.SFFullNode)
	if peer == nil {
		log.WithFields(log.Fields{"module": logModule}).Debug("can't find sync peer")
		return nil
	}

	if peer.Height() < fs.chain.BestBlockHeight()+minGapStartFastSync {
		log.WithFields(log.Fields{"module": logModule}).Debug("Height gap does not meet fast synchronization condition")
		return nil
	}

	fs.syncPeer = peer
	fs.initFastSyncParameters()
	err := fs.findFastSyncRange()
	if err != nil {
		return err
	}

	for {
		blocks, err := fs.fetchBlocks(fs.commonAncestor, fs.stopHeader)
		if err != nil {
			fs.peers.ErrorHandler(peer.ID(), security.LevelConnException, err)
			log.WithFields(log.Fields{"module": logModule, "error": err}).Error("failed on fetch blocks")
			return err
		}

		if err := fs.verifyBlocks(blocks); err != nil {
			fs.peers.ErrorHandler(peer.ID(), security.LevelMsgIllegal, err)
			log.WithFields(log.Fields{"module": logModule, "error": err}).Error("failed on process blocks")
			return err
		}

		if fs.chain.BestBlockHeight() >= fs.stopHeader.Height {
			log.WithFields(log.Fields{"module": logModule, "height": fs.chain.BestBlockHeight()}).Info("fast sync success")
			break
		}

		fs.commonAncestor = fs.chain.BestBlockHeader()
	}

	return nil
}

func (fs *fastSync) fetchBlocks(startHeader *types.BlockHeader, stopHeader *types.BlockHeader) ([]*types.Block, error) {
	startHash := startHeader.Hash()
	stopHash := stopHeader.Hash()
	bodies, err := fs.requireBlocks(fs.syncPeer.ID(), []*bc.Hash{&startHash}, &stopHash)
	if err != nil {
		return nil, err
	}

	return bodies, nil
}

func (fs *fastSync) findCommonAncestor() error {
	headers, err := fs.requireHeaders(fs.syncPeer.ID(), fs.blockLocator(), 1, 0)
	if err != nil {
		return err
	}

	fs.commonAncestor = headers[0]
	return nil
}

func (fs *fastSync) findFastSyncRange() error {
	if err := fs.findCommonAncestor(); err != nil {
		return err
	}

	gap := fs.syncPeer.Height() - uint64(fastSyncPivotGap) - fs.commonAncestor.Height
	if gap > maxFastSyncBlocksNum {
		fs.length = maxFastSyncBlocksNum
	} else {
		fs.length = gap
	}

	startPoint := fs.commonAncestor.Hash()
	headers, err := fs.requireHeaders(fs.syncPeer.ID(), []*bc.Hash{&startPoint}, 2, fs.length-1)
	if err != nil {
		return err
	}

	fs.stopHeader = headers[1]
	return nil
}

func (fs *fastSync) initFastSyncParameters() {
	fs.commonAncestor = nil
	fs.stopHeader = nil
	fs.length = 0
}

func (fs *fastSync) locateBlocks(locator []*bc.Hash, stopHash *bc.Hash) ([]*types.Block, error) {
	headers, err := fs.locateHeaders(locator, stopHash, 0, 0, uint64(maxBlocksPerMsg))
	if err != nil {
		return nil, err
	}

	blocks := []*types.Block{}
	for _, header := range headers {
		headerHash := header.Hash()
		block, err := fs.chain.GetBlockByHash(&headerHash)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
	}
	return blocks, nil
}

func (fs *fastSync) locateHeaders(locator []*bc.Hash, stopHash *bc.Hash, amount uint64, skip uint64, maxNum uint64) ([]*types.BlockHeader, error) {
	startHeader, err := fs.chain.GetHeaderByHeight(0)
	if err != nil {
		return nil, err
	}

	for _, hash := range locator {
		header, err := fs.chain.GetHeaderByHash(hash)
		if err == nil && fs.chain.InMainChain(header.Hash()) {
			startHeader = header
			break
		}
	}

	var stopHeader *types.BlockHeader
	if stopHash != nil {
		stopHeader, err = fs.chain.GetHeaderByHash(stopHash)
	} else {
		stopHeader, err = fs.chain.GetHeaderByHeight(startHeader.Height + uint64((amount-1)*(skip+1)))
	}
	if err != nil {
		return nil, err
	}

	headers := []*types.BlockHeader{}
	num := uint64(0)
	for i := startHeader.Height; i <= stopHeader.Height && num < maxNum; i += uint64(skip) + 1 {
		header, err := fs.chain.GetHeaderByHeight(i)
		if err != nil {
			return nil, err
		}
		headers = append(headers, header)
		num++
	}
	return headers, nil
}

func (fs *fastSync) processBlocks(peerID string, blocks []*types.Block) {
	fs.blocksProcessCh <- &blocksMsg{blocks: blocks, peerID: peerID}
}

func (fs *fastSync) processHeaders(peerID string, headers []*types.BlockHeader) {
	fs.headersProcessCh <- &headersMsg{headers: headers, peerID: peerID}
}

func (fs *fastSync) requireBlocks(peerID string, locator []*bc.Hash, stopHash *bc.Hash) ([]*types.Block, error) {
	peer := fs.peers.GetPeer(peerID)
	if peer == nil {
		return nil, errPeerDropped
	}

	if ok := peer.GetBlocks(locator, stopHash); !ok {
		return nil, errPeerDropped
	}

	timeout := time.NewTimer(syncTimeout)
	defer timeout.Stop()

	for {
		select {
		case msg := <-fs.blocksProcessCh:
			if msg.peerID != peerID {
				continue
			}

			return msg.blocks, nil
		case <-timeout.C:
			return nil, errors.Wrap(errRequestTimeout, "requireBlocks")
		}
	}
}

func (fs *fastSync) requireHeaders(peerID string, locator []*bc.Hash, amount uint64, skip uint64) ([]*types.BlockHeader, error) {
	peer := fs.peers.GetPeer(peerID)
	if peer == nil {
		return nil, errPeerDropped
	}

	if ok := peer.GetHeaders(locator, amount, skip); !ok {
		return nil, errPeerDropped
	}

	timeout := time.NewTimer(syncTimeout)
	defer timeout.Stop()

	for {
		select {
		case msg := <-fs.headersProcessCh:
			if msg.peerID != peerID {
				continue
			}

			if len(msg.headers) != int(amount) {
				return nil, errHeadersNum
			}

			return msg.headers, nil
		case <-timeout.C:
			return nil, errors.Wrap(errRequestTimeout, "requireHeaders")
		}
	}
}

func (fs *fastSync) setSyncPeer(peer *peers.Peer) {
	fs.syncPeer = peer
}

func (fs *fastSync) verifyBlocks(blocks []*types.Block) error {
	for _, block := range blocks {
		isOrphan, err := fs.chain.ProcessBlock(block)
		if err != nil {
			return err
		}

		if isOrphan {
			log.WithFields(log.Fields{"module": logModule, "height": block.Height, "hash": block.Hash()}).Warn("fast sync block is orphan")
		}
	}

	return nil
}
