package chainmgr

import (
	"sync"

	log "github.com/sirupsen/logrus"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"

	"github.com/vapor/netsync/peers"
	"github.com/vapor/p2p/security"
)

type BlockProcessor interface {
	process(chan bool, chan bool, *sync.WaitGroup)
	resetParameter()
}

type downloadedBlock struct {
	startHeight uint64
	stopHeight  uint64
}

type blockProcessor struct {
	chain             Chain
	storage           Storage
	peers             *peers.PeerSet
	downloadedBlockCh chan *downloadedBlock
	queue             *prque.Prque
}

func newBlockProcessor(chain Chain, storage Storage, peers *peers.PeerSet, downloadedBlockCh chan *downloadedBlock) *blockProcessor {
	return &blockProcessor{
		chain:   chain,
		peers:   peers,
		storage: storage,
		queue:   prque.New(),

		downloadedBlockCh: downloadedBlockCh,
	}
}

func (bp *blockProcessor) add(download *downloadedBlock) {
	for i := download.startHeight; i <= download.stopHeight; i++ {
		bp.queue.Push(i, -float32(i))
	}
}

func (bp *blockProcessor) insert(height uint64) error {
	blockStore, err := bp.storage.ReadBlock(height)
	if err != nil {
		return err
	}

	isOrphan, err := bp.chain.ProcessBlock(blockStore.block)
	if err != nil || isOrphan {
		bp.peers.ProcessIllegal(blockStore.peerID, security.LevelMsgIllegal, err.Error())
		return err
	}

	bp.storage.DeleteBlock(height)
	return nil
}

func (bp *blockProcessor) process(downloadComplete chan bool, ProcessComplete chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		for !bp.queue.Empty() {
			height := bp.queue.PopItem().(uint64)
			if height > bp.chain.BestBlockHeight()+1 {
				bp.queue.Push(height, -float32(height))
				break
			}

			if err := bp.insert(height); err != nil {
				ProcessComplete <- true
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("failed on process block")
				return
			}
		}

		select {
		case blocks := <-bp.downloadedBlockCh:
			bp.add(blocks)
			for len(bp.downloadedBlockCh) > 0 {
				bp.add(<-bp.downloadedBlockCh)
			}

		case <-downloadComplete:
			return
		}
	}
}

func (bp *blockProcessor) resetParameter() {
	bp.queue.Reset()
}
