package chainmgr

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"

	"fmt"
	"github.com/vapor/errors"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/p2p/security"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	blockProcessChSize   = 1024
	blocksProcessChSize  = 128
	headersProcessChSize = 1024
)

var (
	requireBlockTimeout   = 20 * time.Second
	requireHeadersTimeout = 30 * time.Second
	requireBlocksTimeout  = 50 * time.Second
	fastSyncTimeout       = 200 * time.Second
)

type MsgFetcher interface {
	resetParameter()
	requireBlock(peerID string, height uint64) (*types.Block, error)
	parallelFetchBlocks(taskQueue *prque.Prque, downloadedBlockCh chan *downloadedBlock, downloadResult chan bool, ProcessResult chan bool, wg *sync.WaitGroup, num int)
	parallelFetchHeaders(peers []*peers.Peer, locator []*bc.Hash, stopHash *bc.Hash, skip uint64) (map[string][]*types.BlockHeader, error)
}

type msgFetcher struct {
	storage          Storage
	peers            *peers.PeerSet
	blockProcessCh   chan *blockMsg
	blocksProcessCh  chan *blocksMsg
	headersProcessCh chan *headersMsg
}

func newMsgFetcher(storage Storage, peers *peers.PeerSet) *msgFetcher {
	return &msgFetcher{
		storage:          storage,
		peers:            peers,
		blockProcessCh:   make(chan *blockMsg, blockProcessChSize),
		blocksProcessCh:  make(chan *blocksMsg, blocksProcessChSize),
		headersProcessCh: make(chan *headersMsg, headersProcessChSize),
	}
}

func (mf *msgFetcher) processBlock(peerID string, block *types.Block) {
	mf.blockProcessCh <- &blockMsg{block: block, peerID: peerID}
}

func (mf *msgFetcher) processBlocks(peerID string, blocks []*types.Block) {
	mf.blocksProcessCh <- &blocksMsg{blocks: blocks, peerID: peerID}
}

func (mf *msgFetcher) processHeaders(peerID string, headers []*types.BlockHeader) {
	mf.headersProcessCh <- &headersMsg{headers: headers, peerID: peerID}
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

type timeoutTime struct {
	time   time.Time
	peerID string
}

func (mf *msgFetcher) parallelFetchBlocks(taskQueue *prque.Prque, downloadedBlockCh chan *downloadedBlock, downloadComplete chan bool, ProcessComplete chan bool, wg *sync.WaitGroup, num int) {
	//wg.Add(1)
	defer fmt.Println("parallelFetchBlocks done. num:", num)
	defer wg.Done()

	//timeout := time.NewTimer(requireBlocksTimeout)
	timeout := time.NewTimer(requireBlocksTimeout)
	defer timeout.Stop()
	fastSyncTimeout := time.NewTimer(fastSyncTimeout)
	defer fastSyncTimeout.Stop()

	tasks := make(map[string]*task)
	stopTimers := []*timeoutTime{}
	for {
		// schedule task
		if taskQueue.Size() == 0 && len(tasks) == 0 {
			downloadComplete <- true
			return
		}

		for !taskQueue.Empty() {
			piece := taskQueue.PopItem().(*piece)
			peerID, err := mf.peers.SelectPeer(piece.stopHeader.Height + fastSyncPivotGap)
			if err != nil {
				if len(tasks) == 0 {
					downloadComplete <- true
					return
				}
				taskQueue.Push(piece, -float32(piece.index))
				log.WithFields(log.Fields{"module": logModule, "err": err}).Debug("failed on select valid peer")
				break
			}

			startHash := piece.startHeader.Hash()
			stopHash := piece.stopHeader.Hash()
			if err := mf.requireBlocks(peerID, []*bc.Hash{&startHash}, &stopHash); err != nil {
				taskQueue.Push(piece, -float32(piece.index))
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("failed on send require blocks msg")
				continue
			}

			tasks[peerID] = &task{piece: piece, startTime: time.Now()}
			stopTimers = append(stopTimers, &timeoutTime{time: time.Now().Add(requireBlocksTimeout), peerID: peerID})
			if len(tasks) == 1 {
				timeout.Reset(requireBlocksTimeout)
			}
		}

		select {
		case msg := <-mf.blocksProcessCh:
			mf.peers.SetIdle(msg.peerID)

			//check message from the requested peer.
			task, ok := tasks[msg.peerID]
			if !ok {
				mf.peers.ErrorHandler(msg.peerID, security.LevelMsgIllegal, errors.New("get unsolicited blocks msg"))
				break
			}

			//reset timeout
			for i, stopTimer := range stopTimers {
				if stopTimer.peerID == msg.peerID {
					stopTimers = append(stopTimers[:i], stopTimers[i+1:]...)
					if i == 0 {
						if len(stopTimers) > 0 {
							timeout.Reset(stopTimers[0].time.Sub(time.Now()))
						}
					}
					break
				}
			}

			if len(msg.blocks) == 0 {
				mf.peers.ErrorHandler(msg.peerID, security.LevelMsgIllegal, errors.New("null blocks msg"))
				taskQueue.Push(task.piece, -float32(task.piece.index))
				break
			}

			// blocks more than request
			if uint64(len(msg.blocks)) > task.piece.stopHeader.Height-task.piece.startHeader.Height+1 {
				mf.peers.ErrorHandler(msg.peerID, security.LevelMsgIllegal, errors.New("exceed length blocks msg"))
				taskQueue.Push(task.piece, -float32(task.piece.index))
				break
			}

			// verify start block
			if msg.blocks[0].Hash() != task.piece.startHeader.Hash() {
				mf.peers.ErrorHandler(msg.peerID, security.LevelMsgIllegal, errors.New("get mismatch blocks msg"))
				taskQueue.Push(task.piece, -float32(task.piece.index))
				break
			}

			// verify blocks continuity
			for i := 0; i < len(msg.blocks)-1; i++ {
				if msg.blocks[i].Hash() != msg.blocks[i+1].PreviousBlockHash {
					mf.peers.ErrorHandler(msg.peerID, security.LevelMsgIllegal, errors.New("get discontinuous blocks msg"))
					taskQueue.Push(task.piece, -float32(task.piece.index))
					break
				}
			}

			if err := mf.storage.WriteBlocks(msg.peerID, msg.blocks); err != nil {
				log.WithFields(log.Fields{"module": logModule, "error": err}).Info("write block error")
				downloadComplete <- true
				return
			}

			downloadedBlockCh <- &downloadedBlock{startHeight: msg.blocks[0].Height, stopHeight: msg.blocks[len(msg.blocks)-1].Height}
			delete(tasks, msg.peerID)
			//unfinished task, continue
			if msg.blocks[len(msg.blocks)-1].Height < task.piece.stopHeader.Height-1 {
				log.WithFields(log.Fields{"module": logModule, "task": task.piece.index}).Info("task unfinished")
				piece := *task.piece
				piece.startHeader = &msg.blocks[len(msg.blocks)-1].BlockHeader
				taskQueue.Push(task.piece, -float32(task.piece.index))
			}
		case <-timeout.C:
			if len(stopTimers) == 0 {
				break
			}

			task, ok := tasks[stopTimers[0].peerID]
			if !ok {
				break
			}
			log.WithFields(log.Fields{"module": logModule, "error": errRequestTimeout}).Info("failed on fetch blocks")
			mf.peers.ErrorHandler(stopTimers[0].peerID, security.LevelConnException, errors.New("require blocks timeout"))
			taskQueue.Push(task.piece, -float32(task.piece.index))
			stopTimers = stopTimers[1:]
			//reset timeout
			if len(stopTimers) > 0 {
				timeout.Reset(stopTimers[0].time.Sub(time.Now()))
			}

			//downloadResult <- false
			//log.WithFields(log.Fields{"module": logModule, "error": errRequestTimeout}).Info("failed on fetch blocks")
			//return
		case <-fastSyncTimeout.C:
			downloadComplete <- true
			return
		case <-ProcessComplete:
			return
		}
	}
}

func (mf *msgFetcher) parallelFetchHeaders(peers []*peers.Peer, locator []*bc.Hash, stopHash *bc.Hash, skip uint64) (map[string][]*types.BlockHeader, error) {
	result := make(map[string][]*types.BlockHeader)

	for _, peer := range peers {
		go peer.GetHeaders(locator, stopHash, skip)
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

func (mf *msgFetcher) resetParameter() {
	for len(mf.blocksProcessCh) > 0 {
		<-mf.blocksProcessCh
	}
	for len(mf.headersProcessCh) > 0 {
		<-mf.headersProcessCh
	}
	mf.storage.resetParameter()
}
