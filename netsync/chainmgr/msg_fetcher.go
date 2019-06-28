package chainmgr

import (
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"

	"github.com/vapor/errors"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	blockProcessChSize   = 1024
	blocksProcessChSize  = 128
	headersProcessChSize = 1024

	fetchDataParallelTimeout = 100 * time.Second
)

type msgFetcher struct {
	peers *peers.PeerSet

	blockProcessCh   chan *blockMsg
	blocksProcessCh  chan *blocksMsg
	headersProcessCh chan *headersMsg
}

func newMsgFetcher(peers *peers.PeerSet) *msgFetcher {
	return &msgFetcher{
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

	timeout := time.NewTimer(syncTimeout)
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

func (mf *msgFetcher) parallelRequireBlocks(taskQueue *prque.Prque) error {
	timeout := time.NewTimer(fetchDataParallelTimeout)
	defer timeout.Stop()

	tasks := make(map[string]*task)
	// schedule task
	for {
		for !taskQueue.Empty() {
			piece := taskQueue.PopItem().(*piece)
			peerID, err := mf.peers.SelectPeer(piece.stopHeader.Height + fastSyncPivotGap)
			if err != nil {
				taskQueue.Push(piece, -float32(piece.index))
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("failed on select valid peer")
				break
			}
			//go fetch(resultCh, task, peerID)
			startHash := piece.startHeader.Hash()
			stopHash := piece.stopHeader.Hash()
			if err := mf.requireBlocks(peerID, []*bc.Hash{&startHash}, &stopHash); err != nil {
				taskQueue.Push(piece, -float32(piece.index))
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("failed on send require blocks msg")
			}

			tasks[startHash.String()] = &task{piece: piece, peerID: peerID, status: progress, startTime: time.Now()}
		}

		select {
		case msg := <-mf.blocksProcessCh:
			if len(msg.blocks) == 0 {
				log.WithFields(log.Fields{"module": logModule}).Error("failed on get null blocks msg")
				//todo: err handle
				break
			}

			msgStart := msg.blocks[0]
			startHash := msgStart.Hash()

			task, ok := tasks[startHash.String()]
			if !ok {
				log.WithFields(log.Fields{"module": logModule}).Error("failed on get unsolicited blocks msg")
				//todo: err handle
				break
			}

			if task.peerID != msg.peerID {
				log.WithFields(log.Fields{"module": logModule, "peerID": msg.peerID}).Error("get blocks msg from wrong peer")
				//todo: err handle
				break
			}

			// blocks more than request
			if uint64(len(msg.blocks)) > task.piece.stopHeader.Height-task.piece.startHeader.Height+1 {
				taskQueue.Push(task.piece, -float32(task.piece.index))
				log.WithFields(log.Fields{"module": logModule}).Error("failed on get null blocks msg")
				break
			}

			// verify blocks msg
			if msgStart.Hash() != task.piece.startHeader.Hash() {
				log.WithFields(log.Fields{"module": logModule, "peerID": msg.peerID}).Error("get mismatch blocks msg from peer")
				//todo: err handle
				break

			}

			// verify blocks msg
			for i := 0; i < len(msg.blocks)-1; i++ {
				if msg.blocks[i].Hash() != msg.blocks[i+1].PreviousBlockHash {
					//todo: peer error handle
					taskQueue.Push(task.piece, -float32(task.piece.index))
					log.WithFields(log.Fields{"module": logModule, "peerID": msg.peerID}).Error("get discontinuous blocks msg from peer")

					break
				}
			}

			//unfinished task, continue
			if msg.blocks[len(msg.blocks)-1].Hash() != task.piece.stopHeader.PreviousBlockHash {
				log.WithFields(log.Fields{"module": logModule, "task": task.piece.index}).Info("task unfinished")

				//todo: send blocks to process module
				piece := *task.piece
				piece.startHeader = &msg.blocks[len(msg.blocks)-1].BlockHeader
				taskQueue.Push(task.piece, -float32(task.piece.index))
				mf.peers.SetIdle(msg.peerID)
				break
			}

			task.status = completion
			mf.peers.SetIdle(msg.peerID)
		case <-timeout.C:
			return errRequestTimeout
		}

	}

	return nil

}

func (mf *msgFetcher) parallelRequireHeaders(peers []*peers.Peer, locator []*bc.Hash, stopHash *bc.Hash, skip uint64) (map[string][]*types.BlockHeader, error) {
	result := make(map[string][]*types.BlockHeader)

	for _, peer := range peers {
		go peer.GetHeaders(locator, stopHash, skip)
	}

	timeout := time.NewTimer(syncTimeout)
	defer timeout.Stop()

	for {
		select {
		case msg := <-mf.headersProcessCh:
			for _, peer := range peers {
				if peer.ID() == msg.peerID {
					result[msg.peerID] = append(result[msg.peerID], msg.headers[:]...)
				}

				if len(result) == len(peers) {
					return result, nil
				}
			}
			log.WithFields(log.Fields{"module": logModule, "peerID": msg.peerID}).Warn("received unsolicited block headers information")

		case <-timeout.C:
			return nil, errors.Wrap(errRequestTimeout, "requireHeaders")
		}
	}
}
