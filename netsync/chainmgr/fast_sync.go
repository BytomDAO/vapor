package chainmgr

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/errors"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

var (
	maxNumOfSkeletonPerSync = uint64(10)
	numOfBlocksSkeletonGap  = maxNumOfBlocksPerMsg
	maxNumOfBlocksPerSync   = numOfBlocksSkeletonGap * maxNumOfSkeletonPerSync
	fastSyncPivotGap        = uint64(64)
	minGapStartFastSync     = uint64(128)

	errNoSyncPeer = errors.New("can't find sync peer")
)

type fastSync struct {
	chain           Chain
	msgFetcher      MsgFetcher
	blockProcessor  BlockProcessor
	length          uint64
	blockFetchTasks []*fetchBlocksWork
	peers           *peers.PeerSet
	mainSyncPeer    *peers.Peer
	stopHeader      *types.BlockHeader
	quite           chan struct{}
}

func newFastSync(chain Chain, msgFetcher MsgFetcher, storage Storage, peers *peers.PeerSet) *fastSync {
	return &fastSync{
		chain:           chain,
		msgFetcher:      msgFetcher,
		blockProcessor:  newBlockProcessor(chain, storage, peers),
		peers:           peers,
		blockFetchTasks: make([]*fetchBlocksWork, 0),
		quite:           make(chan struct{}),
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
		fs.blockFetchTasks = append(fs.blockFetchTasks, &fetchBlocksWork{startHeader: skeleton[i], stopHeader: skeleton[i+1]})
	}

	return nil
}

func (fs *fastSync) process() error {
	if err := fs.findSyncRange(); err != nil {
		return err
	}

	if err := fs.createFetchBlocksTasks(); err != nil {
		return err
	}

	//downloadNotifyCh true:new blocks false:download stop
	downloadNotifyCh := make(chan bool, maxNumOfBlocksPerSync+1)
	processStop := make(chan bool, 1)

	var wg sync.WaitGroup
	wg.Add(2)
	go fs.msgFetcher.parallelFetchBlocks(fs.blockFetchTasks, downloadNotifyCh, processStop, &wg)
	go fs.blockProcessor.process(downloadNotifyCh, processStop, &wg)
	wg.Wait()
	close(processStop)
	close(downloadNotifyCh)
	fs.resetParameter()
	log.WithFields(log.Fields{"module": logModule, "height": fs.chain.BestBlockHeight()}).Info("fast sync complete")
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
	skeletonMap, err := fs.msgFetcher.parallelFetchHeaders(peers, fs.blockLocator(), &stopHash, numOfBlocksSkeletonGap-1)
	if err != nil && len(skeletonMap) == 0 {
		return nil, err
	}

	mainSkeleton, ok := skeletonMap[fs.mainSyncPeer.ID()]
	if !ok || len(mainSkeleton) == 0 {
		return nil, errors.New("No main skeleton found")
	}

	fs.msgFetcher.addSyncPeer(fs.mainSyncPeer.ID())
	delete(skeletonMap, fs.mainSyncPeer.ID())
	for peerID, skeleton := range skeletonMap {
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
		fs.msgFetcher.addSyncPeer(peerID)
	}

	return mainSkeleton, nil
}

// findSyncRange find the start and end of this sync.
// sync length cannot be greater than maxFastSyncBlocksNum.
func (fs *fastSync) findSyncRange() error {
	bestHeight := fs.chain.BestBlockHeight()
	fs.length = fs.mainSyncPeer.IrreversibleHeight() - fastSyncPivotGap - bestHeight
	if fs.length > maxNumOfBlocksPerSync {
		fs.length = maxNumOfBlocksPerSync
	}

	stopBlock, err := fs.msgFetcher.requireBlock(fs.mainSyncPeer.ID(), bestHeight+fs.length)
	if err != nil {
		return err
	}

	fs.stopHeader = &stopBlock.BlockHeader
	return nil
}

func (fs *fastSync) resetParameter() {
	fs.blockFetchTasks = make([]*fetchBlocksWork, 0)
	fs.msgFetcher.resetParameter()
}

func (fs *fastSync) setSyncPeer(peer *peers.Peer) {
	fs.mainSyncPeer = peer
}
