package chainmgr

import (
	"math/rand"
	"sync"

	log "github.com/sirupsen/logrus"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"

	"github.com/vapor/errors"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

var (
	maxBlocksPerMsg      = uint64(500)
	maxHeadersPerMsg     = uint64(500)
	fastSyncPivotGap     = uint64(64)
	minGapStartFastSync  = uint64(128)
	maxFastSyncBlocksNum = uint64(10000)

	errOrphanBlock = errors.New("orphan block found during fast sync")
	errNoSyncPeer  = errors.New("can't find sync peer")
)

type piece struct {
	index                   int
	startHeader, stopHeader *types.BlockHeader
}

type fastSync struct {
	chain             Chain
	msgFetcher        MsgFetcher
	blockProcessor    BlockProcessor
	peers             *peers.PeerSet
	syncPeer          *peers.Peer
	stopHeader        *types.BlockHeader
	length            uint64
	pieces            *prque.Prque
	downloadedBlockCh chan *downloadedBlock
	downloadResult    chan bool
	processResult     chan bool
	quite             chan struct{}
}

func newFastSync(chain Chain, msgFether MsgFetcher, storage Storage, peers *peers.PeerSet) *fastSync {
	downloadedBlockCh := make(chan *downloadedBlock, maxFastSyncBlocksNum)

	return &fastSync{
		chain:             chain,
		blockProcessor:    newBlockProcessor(chain, storage, peers, downloadedBlockCh),
		msgFetcher:        msgFether,
		peers:             peers,
		pieces:            prque.New(),
		downloadedBlockCh: downloadedBlockCh,
		downloadResult:    make(chan bool, 1),
		processResult:     make(chan bool, 1),
		quite:             make(chan struct{}),
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

// createFetchBlocksTasks get the skeleton and assign tasks according to the skeleton.
func (fs *fastSync) createFetchBlocksTasks() error {
	// skeleton is a batch of block headers separated by maxBlocksPerMsg distance.
	skeleton, err := fs.createSkeleton()
	if err != nil {
		return err
	}

	// low height block has high download priority
	for i := 0; i < len(skeleton)-1; i++ {
		fs.pieces.Push(&piece{index: i, startHeader: skeleton[i], stopHeader: skeleton[i+1]}, -float32(i))
	}

	return nil
}

func (fs *fastSync) process() error {
	num := rand.Int()
	log.WithFields(log.Fields{"module": logModule, "num": num, "height": fs.chain.BestBlockHeight()}).Info("fast sync start")
	fs.resetParameter()
	if err := fs.findSyncRange(); err != nil {
		return err
	}

	if err := fs.createFetchBlocksTasks(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go fs.msgFetcher.parallelFetchBlocks(fs.pieces, fs.downloadedBlockCh, fs.downloadResult, fs.processResult, &wg, num)
	go fs.blockProcessor.process(fs.downloadResult, fs.processResult, &wg, num)
	wg.Wait()
	log.WithFields(log.Fields{"module": logModule, "num": num, "height": fs.chain.BestBlockHeight()}).Info("fast sync complete")
	return nil
}

// createSkeleton
func (fs *fastSync) createSkeleton() ([]*types.BlockHeader, error) {
	// Find peers that meet the height requirements.
	peers := fs.peers.GetPeersByHeight(fs.stopHeader.Height + fastSyncPivotGap)
	if len(peers) == 0 {
		return nil, errNoSyncPeer
	}

	// parallel fetch the skeleton from peers.
	stopHash := fs.stopHeader.Hash()
	skeletonMap, err := fs.msgFetcher.parallelFetchHeaders(peers, fs.blockLocator(), &stopHash, maxBlocksPerMsg-1)
	if err != nil {
		return nil, err
	}

	// skeleton 2/3 peer consistent verification.
	mainSkeleton, ok := skeletonMap[fs.syncPeer.ID()]
	if !ok || len(mainSkeleton) == 0 {
		return nil, errors.New("No main skeleton found")
	}

	num := 0
	delete(skeletonMap, fs.syncPeer.ID())
	for _, skeleton := range skeletonMap {
		if len(skeleton) != len(mainSkeleton) {
			log.WithFields(log.Fields{"module": logModule, "main skeleton": len(mainSkeleton), "got skeleton": len(skeleton)}).Warn("different skeleton length")
			continue
		}

		for i, header := range skeleton {
			if header.Hash() != mainSkeleton[i].Hash() {
				log.WithFields(log.Fields{"module": logModule, "header index": i, "main skeleton": mainSkeleton[i].Hash(), "got skeleton": header.Hash()}).Warn("different skeleton hash")
				continue
			}
		}

		num++
	}

	if num < len(peers)*2/3 {
		return nil, errors.New("skeleton consistent verification error")
	}

	return mainSkeleton, nil
}

// findSyncRange find the start and end of this sync.
// sync length cannot be greater than maxFastSyncBlocksNum.
func (fs *fastSync) findSyncRange() error {
	bestHeight := fs.chain.BestBlockHeight()
	fs.length = fs.syncPeer.IrreversibleHeight() - fastSyncPivotGap - bestHeight
	if fs.length > maxFastSyncBlocksNum {
		fs.length = maxFastSyncBlocksNum
	}

	stopBlock, err := fs.msgFetcher.requireBlock(fs.syncPeer.ID(), bestHeight+fs.length)
	if err != nil {
		return err
	}

	fs.stopHeader = &stopBlock.BlockHeader
	return nil
}

func (fs *fastSync) locateBlocks(locator []*bc.Hash, stopHash *bc.Hash) ([]*types.Block, error) {
	headers, err := fs.locateHeaders(locator, stopHash, 0, maxBlocksPerMsg)
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

func (fs *fastSync) locateHeaders(locator []*bc.Hash, stopHash *bc.Hash, skip uint64, maxNum uint64) ([]*types.BlockHeader, error) {
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

	headers := make([]*types.BlockHeader, 0)
	stopHeader, err := fs.chain.GetHeaderByHash(stopHash)
	if err != nil {
		return headers, nil
	}

	if !fs.chain.InMainChain(*stopHash) {
		return headers, nil
	}

	num := uint64(0)
	for i := startHeader.Height; i <= stopHeader.Height && num < maxNum; i += skip + 1 {
		header, err := fs.chain.GetHeaderByHeight(i)
		if err != nil {
			return nil, err
		}

		headers = append(headers, header)
		num++
	}

	return headers, nil
}

func (fs *fastSync) resetParameter() {
	fs.pieces.Reset()
	for _, ch := range []chan bool{fs.downloadResult, fs.processResult} {
		select {
		case <-ch:
		default:
		}
	}

	for len(fs.downloadedBlockCh) > 0 {
		<-fs.downloadedBlockCh
	}

	fs.msgFetcher.resetParameter()
	fs.blockProcessor.resetParameter()
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
			log.WithFields(log.Fields{"module": logModule, "height": block.Height, "hash": block.Hash()}).Error("fast sync block is orphan")
			return errOrphanBlock
		}
	}

	return nil
}
