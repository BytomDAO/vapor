package blockproposer

import (
	"encoding/hex"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/account"
	"github.com/vapor/config"
	"github.com/vapor/event"
	"github.com/vapor/proposal"
	"github.com/vapor/protocol"
)

const (
	logModule = "blockproposer"
)

// BlockProposer propose several block in specified time range
type BlockProposer struct {
	sync.Mutex
	chain           *protocol.Chain
	accountManager  *account.Manager
	txPool          *protocol.TxPool
	started         bool
	quit            chan struct{}
	eventDispatcher *event.Dispatcher
}

// generateBlocks is a worker that is controlled by the proposeWorkerController.
// It is self contained in that it creates block templates and attempts to solve
// them while detecting when it is performing stale work and reacting
// accordingly by generating a new block template.  When a block is verified, it
// is submitted.
//
// It must be run as a goroutine.
func (b *BlockProposer) generateBlocks() {
	xpub := config.CommonConfig.PrivateKey().XPub()
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()
out:
	for {
		select {
		case <-b.quit:
			break out
		case <-ticker.C:
		}

		bestBlockHeader := b.chain.BestBlockHeader()
		bestBlockHash := bestBlockHeader.Hash()
		timeStart, timeEnd, err := b.chain.GetBBFT().NextLeaderTimeRange(xpub[:], &bestBlockHash)
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "error": err, "pubKey": hex.EncodeToString(xpub[:])}).Debug("fail on get next leader time range")
			continue
		}

		now := uint64(time.Now().UnixNano() / 1e6)
		if timeStart < now {
			timeStart = now
		}

		time.Sleep(time.Millisecond * time.Duration(timeStart-now))

		count := 0
		for now = timeStart; now < timeEnd && count < protocol.BlockNumEachNode; now = uint64(time.Now().UnixNano() / 1e6) {
			block, err := proposal.NewBlockTemplate(b.chain, b.txPool, b.accountManager, now)
			if err != nil {
				log.Errorf("failed on create NewBlockTemplate: %v", err)
			} else {
				if isOrphan, err := b.chain.ProcessBlock(block); err == nil {
					log.WithFields(log.Fields{
						"module":   logModule,
						"height":   block.BlockHeader.Height,
						"isOrphan": isOrphan,
						"tx":       len(block.Transactions),
					}).Info("Proposer processed block")

					// Broadcast the block and announce chain insertion event
					if err = b.eventDispatcher.Post(event.NewProposedBlockEvent{Block: *block}); err != nil {
						log.WithFields(log.Fields{"module": logModule, "height": block.BlockHeader.Height, "error": err}).Errorf("Proposer fail on post block")
					}
					count++
				} else {
					log.WithFields(log.Fields{"module": logModule, "height": block.BlockHeader.Height, "error": err}).Errorf("Proposer fail on ProcessBlock")
				}
			}
		}
	}
}

// Start begins the block propose process as well as the speed monitor used to
// track hashing metrics.  Calling this function when the block proposer has
// already been started will have no effect.
//
// This function is safe for concurrent access.
func (b *BlockProposer) Start() {
	b.Lock()
	defer b.Unlock()

	// Nothing to do if the miner is already running
	if b.started {
		return
	}

	b.quit = make(chan struct{})
	go b.generateBlocks()

	b.started = true
	log.Infof("block proposer started")
}

// Stop gracefully stops the proposal process by signalling all workers, and the
// speed monitor to quit.  Calling this function when the block proposer has not
// already been started will have no effect.
//
// This function is safe for concurrent access.
func (b *BlockProposer) Stop() {
	b.Lock()
	defer b.Unlock()

	// Nothing to do if the miner is not currently running
	if !b.started {
		return
	}

	close(b.quit)
	b.started = false
	log.Info("block proposer stopped")
}

// IsProposing returns whether the block proposer has been started.
//
// This function is safe for concurrent access.
func (b *BlockProposer) IsProposing() bool {
	b.Lock()
	defer b.Unlock()

	return b.started
}

// NewBlockProposer returns a new instance of a block proposer for the provided configuration.
// Use Start to begin the proposal process.  See the documentation for BlockProposer
// type for more details.
func NewBlockProposer(c *protocol.Chain, accountManager *account.Manager, txPool *protocol.TxPool, dispatcher *event.Dispatcher) *BlockProposer {
	return &BlockProposer{
		chain:           c,
		accountManager:  accountManager,
		txPool:          txPool,
		eventDispatcher: dispatcher,
	}
}
