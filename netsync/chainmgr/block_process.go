package chainmgr

import (
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/p2p/security"
)

type BlockProcessor interface {
	process(chan bool, chan bool, *sync.WaitGroup)
}

type blockProcessor struct {
	chain            Chain
	storage          Storage
	peers            *peers.PeerSet
	newBlockNotifyCh chan struct{}
}

func newBlockProcessor(chain Chain, storage Storage, peers *peers.PeerSet, newBlockNotifyCh chan struct{}) *blockProcessor {
	return &blockProcessor{
		chain:            chain,
		peers:            peers,
		storage:          storage,
		newBlockNotifyCh: newBlockNotifyCh,
	}
}

func (bp *blockProcessor) insert(blockStorage *blockStorage) error {
	isOrphan, err := bp.chain.ProcessBlock(blockStorage.block)
	if err != nil || isOrphan {
		bp.peers.ProcessIllegal(blockStorage.peerID, security.LevelMsgIllegal, err.Error())
	}
	return err
}

func (bp *blockProcessor) process(downloadComplete chan bool, ProcessStop chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		for {
			nextHeight := bp.chain.BestBlockHeight() + 1
			block, err := bp.storage.readBlock(nextHeight)
			if err != nil {
				break
			}

			if err := bp.insert(block); err != nil {
				ProcessStop <- true
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("failed on process block")
				return
			}

			bp.storage.deleteBlock(nextHeight)
		}

		select {
		case <-bp.newBlockNotifyCh:
		case <-downloadComplete:
			return
		}
	}
}
