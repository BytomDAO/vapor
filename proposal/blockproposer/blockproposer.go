package blockproposer

import (
	"sync"
	"time"
	"encoding/hex"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/account"
	"github.com/vapor/event"
	"github.com/vapor/proposal"
	"github.com/vapor/protocol"
)

const (
	defaultNumWorkers = 1
	logModule         = "blockproposer"
)

// BlockProposer propose several block in specified time range
type BlockProposer struct {
	sync.Mutex
	chain            *protocol.Chain
	accountManager   *account.Manager
	txPool           *protocol.TxPool
	numWorkers       uint64
	started          bool
	discreteMining   bool
	workerWg         sync.WaitGroup
	updateNumWorkers chan struct{}
	quit             chan struct{}
	eventDispatcher  *event.Dispatcher
}

// generateBlocks is a worker that is controlled by the proposeWorkerController.
// It is self contained in that it creates block templates and attempts to solve
// them while detecting when it is performing stale work and reacting
// accordingly by generating a new block template.  When a block is verified, it
// is submitted.
//
// It must be run as a goroutine.
func (b *BlockProposer) generateBlocks(quit chan struct{}) {
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()
out:
	for {
		select {
		case <-quit:
			break out
		case <-ticker.C:
		}

		bestBlockHeader := b.chain.BestBlockHeader()
		bestBlockHash := bestBlockHeader.Hash()
		var pubKey []byte
		timeStart, timeEnd, err := b.chain.GetBBFT().NextLeaderTimeRange(pubKey, &bestBlockHash)
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "error": err, "pubKey": hex.EncodeToString(pubKey)}).Debug("fail on get next leader time range")
			continue
		}

		now := uint64(time.Now().UnixNano() / 1e6)
		if timeStart < now {
			timeStart = now
		}

		time.Sleep(time.Millisecond * time.Duration(timeStart - now))

		count := 0
		for now = timeStart; now < timeEnd && count < protocol.BlockNumEachNode; now = uint64(time.Now().UnixNano() / 1e6) {
			block, err := proposal.NewBlockTemplate(b.chain, b.txPool, b.accountManager)
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

	b.workerWg.Done()
}

// proposeWorkerController launches the worker goroutines that are used to
// generate block templates.  It also provides the ability to
// dynamically adjust the number of running worker goroutines.
//
// It must be run as a goroutine.
func (b *BlockProposer) proposeWorkerController() {
	// launchWorkers groups common code to launch a specified number of
	// workers for generating blocks.
	var runningWorkers []chan struct{}
	launchWorkers := func(numWorkers uint64) {
		for i := uint64(0); i < numWorkers; i++ {
			quit := make(chan struct{})
			runningWorkers = append(runningWorkers, quit)

			b.workerWg.Add(1)
			go b.generateBlocks(quit)
		}
	}

	// Launch the current number of workers by default.
	runningWorkers = make([]chan struct{}, 0, b.numWorkers)
	launchWorkers(b.numWorkers)

out:
	for {
		select {
		// Update the number of running workers.
		case <-b.updateNumWorkers:
			// No change.
			numRunning := uint64(len(runningWorkers))
			if b.numWorkers == numRunning {
				continue
			}

			// Add new workers.
			if b.numWorkers > numRunning {
				launchWorkers(b.numWorkers - numRunning)
				continue
			}

			// Signal the most recently created goroutines to exit.
			for i := numRunning - 1; i >= b.numWorkers; i-- {
				close(runningWorkers[i])
				runningWorkers[i] = nil
				runningWorkers = runningWorkers[:i]
			}

		case <-b.quit:
			for _, quit := range runningWorkers {
				close(quit)
			}
			break out
		}
	}

	b.workerWg.Wait()
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
	go b.proposeWorkerController()

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

// SetNumWorkers sets the number of workers to create which solve blocks.  Any
// negative values will cause a default number of workers to be used which is
// based on the number of processor cores in the system.  A value of 0 will
// cause all block proposer to be stopped.
//
// This function is safe for concurrent access.
func (b *BlockProposer) SetNumWorkers(numWorkers int32) {
	if numWorkers == 0 {
		b.Stop()
	}

	// Don't lock until after the first check since Stop does its own
	// locking.
	b.Lock()
	defer b.Unlock()

	// Use default if provided value is negative.
	if numWorkers < 0 {
		b.numWorkers = defaultNumWorkers
	} else {
		b.numWorkers = uint64(numWorkers)
	}

	// When the proposer is already running, notify the controller about the
	// the change.
	if b.started {
		b.updateNumWorkers <- struct{}{}
	}
}

// NumWorkers returns the number of workers which are running to solve blocks.
//
// This function is safe for concurrent access.
func (b *BlockProposer) NumWorkers() int32 {
	b.Lock()
	defer b.Unlock()

	return int32(b.numWorkers)
}

// NewBlockProposer returns a new instance of a block proposer for the provided configuration.
// Use Start to begin the proposal process.  See the documentation for BlockProposer
// type for more details.
func NewBlockProposer(c *protocol.Chain, accountManager *account.Manager, txPool *protocol.TxPool, dispatcher *event.Dispatcher) *BlockProposer {
	return &BlockProposer{
		chain:            c,
		accountManager:   accountManager,
		txPool:           txPool,
		numWorkers:       defaultNumWorkers,
		updateNumWorkers: make(chan struct{}),
		eventDispatcher:  dispatcher,
	}
}
