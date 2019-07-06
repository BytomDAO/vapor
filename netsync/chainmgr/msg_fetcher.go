package chainmgr

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/errors"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	maxNumOfParallelFetchBlocks = 7
	blockProcessChSize          = 1024
	blocksProcessChSize         = 128
	headersProcessChSize        = 1024
	maxNumOfRequest             = 3
)

var (
	requireBlockTimeout         = 20 * time.Second
	requireHeadersTimeout       = 30 * time.Second
	requireBlocksTimeout        = 50 * time.Second
	parallelFetchHeadersTimeout = 50 * time.Second
	parallelFetchBlocksTimeout  = 200 * time.Second

	errRequestBlocksTimeout = errors.New("request blocks timeout")
	errRequestBlocks        = errors.New("request blocks err")
)

type MsgFetcher interface {
	resetParameter()
	requireBlock(peerID string, height uint64) (*types.Block, error)
	parallelFetchBlocks(syncPeers []string, work []*fetchBlocksWork, downloadedBlockCh chan *downloadedBlock, downloadResult chan bool, ProcessResult chan bool, wg *sync.WaitGroup, num int)
	parallelFetchHeaders(peers []*peers.Peer, locator []*bc.Hash, stopHash *bc.Hash, skip uint64) (map[string][]*types.BlockHeader, error)
}

type fetchBlocksWork struct {
	index                   int
	startHeader, stopHeader *types.BlockHeader
}

type fetchBlocksResult struct {
	index int
	err   error
}

type msgFetcher struct {
	storage          Storage
	peers            *peers.PeerSet
	blockProcessCh   chan *blockMsg
	blocksProcessCh  chan *blocksMsg
	headersProcessCh chan *headersMsg
	blocksMsgChanMap map[string]chan []*types.Block
}

func newMsgFetcher(storage Storage, peers *peers.PeerSet) *msgFetcher {
	return &msgFetcher{
		storage:          storage,
		peers:            peers,
		blockProcessCh:   make(chan *blockMsg, blockProcessChSize),
		blocksProcessCh:  make(chan *blocksMsg, blocksProcessChSize),
		headersProcessCh: make(chan *headersMsg, headersProcessChSize),
		blocksMsgChanMap: make(map[string]chan []*types.Block),
	}
}

func (mf *msgFetcher) processBlock(peerID string, block *types.Block) {
	mf.blockProcessCh <- &blockMsg{block: block, peerID: peerID}
}

func (mf *msgFetcher) processBlocks(peerID string, blocks []*types.Block) {
	mf.blocksProcessCh <- &blocksMsg{blocks: blocks, peerID: peerID}
	blocksMsgChan, ok := mf.blocksMsgChanMap[peerID]
	if !ok {
		return
	}

	blocksMsgChan <- blocks
}

func (mf *msgFetcher) processHeaders(peerID string, headers []*types.BlockHeader) {
	mf.headersProcessCh <- &headersMsg{headers: headers, peerID: peerID}
}

func (mf *msgFetcher) fetchBlocks(work *fetchBlocksWork) ([]*types.Block, error) {
	peerID, err := mf.peers.SelectPeer(,work.stopHeader.Height + fastSyncPivotGap)
	if err != nil {
		return nil, err
	}

	startHash := work.startHeader.Hash()
	stopHash := work.stopHeader.Hash()
	if err := mf.requireBlocks(peerID, []*bc.Hash{&startHash}, &stopHash); err != nil {
		return nil, err
	}

	timeout := time.NewTimer(requireBlocksTimeout)
	defer timeout.Stop()

	select {
	case blocks := <-mf.blocksMsgChanMap[peerID]:
		//blocks verify
		if err := mf.verifyBlocksMsg(blocks, work.startHeader, work.stopHeader); err != nil {
			return nil, err
		}

		return blocks, nil
	case <-timeout.C:
		return nil, errRequestBlocksTimeout
	}

	return nil, nil
}

func (mf *msgFetcher) fetchBlocksProcess(work *fetchBlocksWork, downloadedBlockCh chan *downloadedBlock) error {
	for countOfRequest := 0; countOfRequest < maxNumOfRequest; countOfRequest++ {
		blocks, err := mf.fetchBlocks(work)
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "work": work.index, "request count": countOfRequest, "error": err}).Info("failed on fetch blocks")
			continue
		}

		// send to block process pool
		downloadedBlockCh <- &downloadedBlock{startHeight: blocks[0].Height, stopHeight: blocks[len(blocks)-1].Height}

		// work completed
		if blocks[len(blocks)-1].Height >= work.stopHeader.Height-1 {
			return nil
		}

		//unfinished work, continue
		work.startHeader = &blocks[len(blocks)-1].BlockHeader
		countOfRequest = 0
	}

	return errRequestBlocks
}

func (mf *msgFetcher) fetchBlocksWorker(workCh chan *fetchBlocksWork, resultCh chan *fetchBlocksResult, closeCh chan struct{}, downloadedBlockCh chan *downloadedBlock, wg *sync.WaitGroup) {
	for {
		select {
		case work := <-workCh:
			err := mf.fetchBlocksProcess(work, downloadedBlockCh)
			resultCh <- &fetchBlocksResult{index: work.index, err: err}
		case <-closeCh:
			wg.Done()
			return
		}
	}
}

func (mf *msgFetcher) parallelFetchBlocks(syncPeers []string, works []*fetchBlocksWork, downloadedBlockCh chan *downloadedBlock, downloadStop chan bool, ProcessComplete chan bool, wg *sync.WaitGroup, num int) {
	defer wg.Done()
	workSize := len(works)
	workCh := make(chan *fetchBlocksWork, workSize)
	resultCh := make(chan *fetchBlocksResult, workSize)
	closeCh := make(chan struct{})

	var workWg sync.WaitGroup
	for i := 0; i <= maxNumOfParallelFetchBlocks && i < workSize; i++ {
		wg.Add(1)
		go mf.fetchBlocksWorker(workCh, resultCh, closeCh, downloadedBlockCh, &workWg)
	}

	for _, work := range works {
		workCh <- work
	}

	//collect fetch results
	for i := 0; i < workSize; i++ {
		result := <-resultCh
		if result.err != nil {
			log.WithFields(log.Fields{"module": logModule, "work": result.index, "err": result.err}).Error("failed on fetch blocks")
			break
		}
	}

	close(closeCh)
	wg.Wait()
	close(workCh)
	close(resultCh)
	downloadStop <- true
}

func (mf *msgFetcher) parallelFetchHeaders(peers []*peers.Peer, locator []*bc.Hash, stopHash *bc.Hash, skip uint64) (map[string][]*types.BlockHeader, error) {
	result := make(map[string][]*types.BlockHeader)

	for _, peer := range peers {
		peer.GetHeaders(locator, stopHash, skip)
	}

	timeout := time.NewTimer(requireHeadersTimeout)
	defer timeout.Stop()

	for {
		select {
		case msg := <-mf.headersProcessCh:
			for _, peer := range peers {
				if peer.ID() == msg.peerID {
					result[msg.peerID] = append(result[msg.peerID], msg.headers[:]...)
					if len(result) == len(peers) {
						return result, nil
					}
					break
				}
			}

		case <-timeout.C:
			return nil, errors.Wrap(errRequestTimeout, "parallelFetchHeaders")
		}
	}
}

func (mf *msgFetcher) requireBlock(peerID string, height uint64) (*types.Block, error) {
	peer := mf.peers.GetPeer(peerID)
	if peer == nil {
		return nil, errPeerDropped
	}

	if ok := peer.GetBlockByHeight(height); !ok {
		return nil, errPeerDropped
	}

	timeout := time.NewTimer(requireBlockTimeout)
	defer timeout.Stop()

	for {
		select {
		case msg := <-mf.blockProcessCh:
			if msg.peerID != peerID {
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

func (mf *msgFetcher) requireBlocks(peerID string, locator []*bc.Hash, stopHash *bc.Hash) error {
	peer := mf.peers.GetPeer(peerID)
	if peer == nil {
		return errPeerDropped
	}

	if ok := peer.GetBlocks(locator, stopHash); !ok {
		return errPeerDropped
	}

	return nil
}

func (mf *msgFetcher) requireHeaders(peerID string, locator []*bc.Hash, stopHash *bc.Hash, skip uint64) error {
	peer := mf.peers.GetPeer(peerID)
	if peer == nil {
		return errPeerDropped
	}

	if ok := peer.GetHeaders(locator, stopHash, skip); !ok {
		return errPeerDropped
	}

	return nil
}

func (mf *msgFetcher) resetParameter() {
	for len(mf.blocksProcessCh) > 0 {
		<-mf.blocksProcessCh
	}

	for len(mf.headersProcessCh) > 0 {
		<-mf.headersProcessCh
	}
	mf.storage.ResetParameter()
}

func (mf *msgFetcher) verifyBlocksMsg(blocks []*types.Block, startHeader, stopHeader *types.BlockHeader) error {
	// null blocks
	if len(blocks) == 0 {
		return errors.New("null blocks msg")
	}

	// blocks more than request
	if uint64(len(blocks)) > stopHeader.Height-startHeader.Height+1 {
		return errors.New("exceed length blocks msg")
	}

	// verify start block
	if blocks[0].Hash() != startHeader.Hash() {
		return errors.New("get mismatch blocks msg")
	}

	// verify blocks continuity
	for i := 0; i < len(blocks)-1; i++ {
		if blocks[i].Hash() != blocks[i+1].PreviousBlockHash {
			return errors.New("get discontinuous blocks msg")
		}
	}

	return nil
}
