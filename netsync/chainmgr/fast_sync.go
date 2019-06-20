package chainmgr

import (
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"

	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

var (
	FastSyncTimeout          = 200 * time.Second
	fetchDataParallelTimeout = 100 * time.Second

	maxFetchRetryNum      = 3
	maxBlockPerMsg        = 100
	maxBlockHeadersPerMsg = 1000
	minGapStartFastSync   = 128
	maxFastSyncBlockNum   = 10000

	errSkeletonMismatch = errors.New("failed on fill the Skeleton")
	errHeadersMismatch  = errors.New("failed on connect block headers")
	errHeadersNum       = errors.New("headers number error")
	errBlocksNum        = errors.New("blocks number error")
	errNoCommonAncestor = errors.New("can't find common ancestor")
	errNoSkeleton       = errors.New("can't find Skeleton")
	errRequireHeaders   = errors.New("require headers err")
	errRequireBlocks    = errors.New("require blocks err")
	errWrongHeaderSize  = errors.New("wrong header size")
	errOrphanBlock      = errors.New("block is orphan")
	errFastSyncTimeout  = errors.New("fast sync timeout")
)

type requireTask struct {
	index       int
	count       int
	length      int
	peerID      string
	startHeader *types.BlockHeader
	stopHeader  *types.BlockHeader
}

type taskResult struct {
	err  error
	task *requireTask
}

type fastSyncResult struct {
	success bool
	err     error
}

func (bk *blockKeeper) blockLocator() []*bc.Hash {
	header := bk.chain.BestBlockHeader()
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
			header, err = bk.chain.GetHeaderByHeight(0)
		} else {
			header, err = bk.chain.GetHeaderByHeight(header.Height - step)
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

func (bk *blockKeeper) fastSynchronize() error {
	err := bk.findFastSyncRange()
	if err != nil {
		return err
	}

	bk.initFastSyncParameters()
	timeout := time.NewTimer(FastSyncTimeout)
	defer timeout.Stop()

	resultCh := make(chan *fastSyncResult, 1)
	go bk.fetchData(resultCh)
	go bk.verifyBlocks(resultCh)

	select {
	case result := <-resultCh:
		if result.err != nil {
			close(bk.fastSyncQuit)
			return err
		}
		return nil
	case <-timeout.C:
		close(bk.fastSyncQuit)
		return errFastSyncTimeout
	}
	return nil
}

func (bk *blockKeeper) fetchBodiesParallel() error {
	return bk.fetchDataParallel(bk.createFetchBodiesTask, bk.fetchBodies, bk.bodiesTaskQueue)
}

func (bk *blockKeeper) fetchBodies(resultCh chan *taskResult, task *requireTask) {
	task.count++
	startHash := task.startHeader.Hash()
	stopHash := task.stopHeader.Hash()
	bodies, err := bk.requireBlocks(task.peerID, []*bc.Hash{&startHash}, &stopHash, task.length)
	if err != nil {
		resultCh <- &taskResult{err: err, task: task}
		return
	}

	bk.bodies = append(bk.bodies[:task.index*maxBlockPerMsg], bodies[:]...)
	bk.blocksProcessIndexCh <- task.index
	resultCh <- &taskResult{err: nil, task: task}
}

func (bk *blockKeeper) createFetchBodiesTask() {
	index := 0
	for i := 0; i < bk.fastSyncLength; i += maxBlockPerMsg {
		var stopHeader *types.BlockHeader
		startHead := bk.headers[i]
		if i+maxBlockPerMsg >= bk.fastSyncLength {
			stopHeader = bk.headers[bk.fastSyncLength-1]
		} else {
			stopHeader = bk.headers[i+maxBlockPerMsg-1]
		}

		bk.bodiesTaskQueue.Push(&requireTask{index: index, length: int(stopHeader.Height - startHead.Height + 1), startHeader: startHead, stopHeader: stopHeader}, -float32(i))
		index++
	}
}

func (bk *blockKeeper) fetchDataParallel(createTask func(), fetch func(chan *taskResult, *requireTask), taskQueue *prque.Prque) error {
	createTask()
	resultCh := make(chan *taskResult, 1)
	taskFinished := 0
	taskSize := taskQueue.Size()
	timeout := time.NewTimer(fetchDataParallelTimeout)
	defer timeout.Stop()

	// schedule task
	for {
		for !taskQueue.Empty() {
			task := taskQueue.PopItem().(*requireTask)
			peerID, err := bk.peers.SelectPeer(bk.skeleton[len(bk.skeleton)-1].Height - uint64(minGapStartFastSync))
			if err != nil {
				taskQueue.Push(task, -float32(task.index))
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("failed on select valid peer")
				break
			}
			task.peerID = peerID
			go fetch(resultCh, task)
		}

		select {
		case result := <-resultCh:
			bk.peers.SetIdle(result.task.peerID)
			if result.err != nil && result.task.count >= maxFetchRetryNum {
				log.WithFields(log.Fields{"module": logModule, "index": result.task.index, "err": result.err}).Error("failed on fetch data")
				return result.err
			}

			if result.err != nil && result.task.count < maxFetchRetryNum {
				log.WithFields(log.Fields{"module": logModule, "count": result.task.count, "index": result.task.index, "err": result.err}).Warn("failed on fetch data")
				taskQueue.Push(result.task, -float32(result.task.index))
				break
			}
			taskFinished++
			if taskFinished == taskSize {
				return nil
			}
		case <-timeout.C:
			return errRequestTimeout
		case <-bk.fastSyncQuit:
			return nil
		}
	}

	return nil
}

func (bk *blockKeeper) fetchHeadersParallel() error {
	return bk.fetchDataParallel(bk.createFetchHeadersTask, bk.fetchHeaders, bk.headersTaskQueue)
}

func (bk *blockKeeper) findCommonAncestor() error {
	headers, err := bk.requireHeaders(bk.syncPeer.ID(), bk.blockLocator(), 1, 0)
	if err != nil {
		return err
	}

	if len(headers) != 1 {
		return errNoCommonAncestor
	}

	bk.commonAncestor = headers[0]
	return nil
}

func (bk *blockKeeper) findFastSyncRange() error {
	if err := bk.findCommonAncestor(); err != nil {
		return err
	}

	gap := bk.syncPeer.Height() - bk.commonAncestor.Height

	if gap > uint64(maxFastSyncBlockNum+minGapStartFastSync) {
		bk.fastSyncLength = maxFastSyncBlockNum
		return nil
	}

	bk.fastSyncLength = int(gap) - minGapStartFastSync
	return nil
}

func (bk *blockKeeper) initFastSyncParameters() {
	bk.headersTaskQueue = prque.New()
	bk.bodiesTaskQueue = prque.New()
	bk.blockProcessQueue = prque.New()
	bk.blocksProcessIndexCh = make(chan int, maxFastSyncBlockNum/maxBlockPerMsg)
	bk.fastSyncQuit = make(chan struct{})
	bk.skeleton = make([]*types.BlockHeader, 0)
	bk.headers = make([]*types.BlockHeader, bk.fastSyncLength)
	bk.bodies = make([]*types.Block, bk.fastSyncLength)
}

func (bk *blockKeeper) locateBlocks(locator []*bc.Hash, stopHash *bc.Hash) ([]*types.Block, error) {
	headers, err := bk.locateHeaders(locator, stopHash, 0, 0)
	if err != nil {
		return nil, err
	}

	blocks := []*types.Block{}
	for i, header := range headers {
		if i >= maxBlockPerMsg {
			break
		}

		headerHash := header.Hash()
		block, err := bk.chain.GetBlockByHash(&headerHash)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
	}
	return blocks, nil
}

func (bk *blockKeeper) locateHeaders(locator []*bc.Hash, stopHash *bc.Hash, amount int, skip int) ([]*types.BlockHeader, error) {
	if amount > maxBlockHeadersPerMsg {
		amount = maxBlockHeadersPerMsg
	}

	startHeader, err := bk.chain.GetHeaderByHeight(0)
	if err != nil {
		return nil, err
	}

	for _, hash := range locator {
		header, err := bk.chain.GetHeaderByHash(hash)
		if err == nil && bk.chain.InMainChain(header.Hash()) {
			startHeader = header
			break
		}
	}

	var stopHeader *types.BlockHeader
	if stopHash != nil {
		stopHeader, err = bk.chain.GetHeaderByHash(stopHash)
	} else {
		stopHeader, err = bk.chain.GetHeaderByHeight(startHeader.Height + uint64((amount-1)*(skip+1)))
	}
	if err != nil {
		return nil, err
	}

	headers := []*types.BlockHeader{}
	num := 0
	for i := startHeader.Height; i <= stopHeader.Height && num < maxBlockHeadersPerMsg; i += uint64(skip) + 1 {
		header, err := bk.chain.GetHeaderByHeight(i)
		if err != nil {
			return nil, err
		}

		headers = append(headers, header)
		num++
	}
	return headers, nil
}

func (bk *blockKeeper) requireBlocks(peerID string, locator []*bc.Hash, stopHash *bc.Hash, length int) ([]*types.Block, error) {
	peer := bk.peers.GetPeer(peerID)
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
		case msg := <-bk.blocksProcessCh:
			if msg.peerID != peerID {
				continue
			}
			if len(msg.blocks) != length {
				return nil, errBlocksNum
			}

			return msg.blocks, nil
		case <-timeout.C:
			return nil, errors.Wrap(errRequestTimeout, "requireBlocks")
		case <-bk.fastSyncQuit:
			return nil, nil
		}
	}
}

func (bk *blockKeeper) requireHeaders(peerID string, locator []*bc.Hash, amount int, skip int) ([]*types.BlockHeader, error) {
	peer := bk.peers.GetPeer(peerID)
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
		case msg := <-bk.headersProcessCh:
			if msg.peerID != peerID {
				continue
			}

			if len(msg.headers) != int(amount) {
				return nil, errHeadersNum
			}

			return msg.headers, nil
		case <-timeout.C:
			return nil, errors.Wrap(errRequestTimeout, "requireHeaders")
		case <-bk.fastSyncQuit:
			return nil, nil
		}
	}
}

func (bk *blockKeeper) createFetchHeadersTask() {
	for index := 0; index < len(bk.skeleton)-1; index++ {
		bk.headersTaskQueue.Push(&requireTask{index: index, length: int(bk.skeleton[index+1].Height - bk.skeleton[index].Height), startHeader: bk.skeleton[index]}, -float32(index))
	}
}

func (bk *blockKeeper) fetchHeaders(resultCh chan *taskResult, task *requireTask) {
	task.count++
	headerHash := task.startHeader.Hash()
	headers, err := bk.requireHeaders(task.peerID, []*bc.Hash{&headerHash}, task.length, 0)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Error("failed on fetch headers")
		resultCh <- &taskResult{err: err, task: task}
		return
	}

	//valid skeleton match
	if headers[len(headers)-1].Hash() != bk.skeleton[task.index+1].PreviousBlockHash {
		log.WithFields(log.Fields{"module": logModule, "error": errSkeletonMismatch}).Error("failed on fetch headers")
		resultCh <- &taskResult{err: errSkeletonMismatch, task: task}
		return
	}

	//valid headers
	for i := 0; i < len(headers)-1; i++ {
		if headers[i+1].PreviousBlockHash != headers[i].Hash() {
			log.WithFields(log.Fields{"module": logModule, "error": errHeadersMismatch}).Error("failed on fetch headers")
			resultCh <- &taskResult{err: errHeadersMismatch, task: task}
			return
		}
	}

	bk.headers = append(bk.headers[:task.index*maxBlockHeadersPerMsg], headers[:]...)
	resultCh <- &taskResult{err: nil, task: task}
}

func (bk *blockKeeper) fetchSkeleton() error {
	startPoint := bk.commonAncestor.Hash()

	bk.skeleton = append(bk.skeleton, bk.commonAncestor)
	if bk.fastSyncLength > maxBlockHeadersPerMsg {
		headers, err := bk.requireHeaders(bk.syncPeer.ID(), []*bc.Hash{&startPoint}, bk.fastSyncLength/maxBlockHeadersPerMsg+1, maxBlockHeadersPerMsg-1)
		if err != nil {
			return err
		}

		bk.skeleton = append(bk.skeleton, headers[1:]...)
	}

	if bk.fastSyncLength%maxBlockHeadersPerMsg != 0 {
		headers, err := bk.requireHeaders(bk.syncPeer.ID(), []*bc.Hash{&startPoint}, 2, bk.fastSyncLength-1)
		if err != nil {
			return err
		}

		bk.skeleton = append(bk.skeleton, headers[1])
	}

	return nil
}

func (bk *blockKeeper) fetchData(result chan *fastSyncResult) {
	if err := bk.fetchSkeleton(); err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Error("failed on fetch skeleton")
		result <- &fastSyncResult{success: false, err: err}
		return
	}

	if err := bk.fetchHeadersParallel(); err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Error("failed on fetch headers parallel")
		result <- &fastSyncResult{success: false, err: err}
		return
	}

	if err := bk.fetchBodiesParallel(); err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Error("failed on fetch bodies")
		result <- &fastSyncResult{success: false, err: err}
		return
	}

	log.WithFields(log.Fields{"module": logModule}).Info("fetch data success")
}

func (bk *blockKeeper) verifyBlocks(result chan *fastSyncResult) {
	processIndex := 0

	for {
		select {
		case index := <-bk.blocksProcessIndexCh:
			bk.blockProcessQueue.Push(nil, -float32(index))
			for !bk.blockProcessQueue.Empty() {
				_, priority := bk.blockProcessQueue.Pop()
				if -priority > float32(processIndex) {
					bk.blockProcessQueue.Push(nil, float32(priority))
					break
				}

				for index := processIndex * maxBlockPerMsg; index < (processIndex+1)*maxBlockPerMsg && index < bk.fastSyncLength; index++ {
					isOrphan, err := bk.chain.ProcessBlock(bk.bodies[index])
					if err != nil {
						result <- &fastSyncResult{success: false, err: err}
					}

					if isOrphan {
						log.WithFields(log.Fields{"module": logModule}).Error("failed on fast sync block is orphan")
						result <- &fastSyncResult{success: false, err: errOrphanBlock}
					}

					if index == bk.fastSyncLength-1 {
						result <- &fastSyncResult{success: true, err: nil}
					}
				}

				processIndex++
			}
		case <-bk.fastSyncQuit:
			return
		}
	}
}
