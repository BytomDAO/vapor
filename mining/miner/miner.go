package miner

import (
	"errors"
	"sync"
	"time"

	"github.com/vapor/config"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/account"
	"github.com/vapor/common"
	"github.com/vapor/consensus"
	engine "github.com/vapor/consensus/consensus"
	"github.com/vapor/consensus/consensus/dpos"
	"github.com/vapor/crypto"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/mining"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm/vmutil"
)

const (
	maxNonce          = ^uint64(0) // 2^64 - 1
	defaultNumWorkers = 1
	hashUpdateSecs    = 1
	module            = "miner"
)

var ConsensusEngine engine.Engine

// Miner creates blocks and searches for proof-of-work values.
type Miner struct {
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
	newBlockCh       chan *bc.Hash
	Authoritys       map[string]string
	position         uint64
	engine           engine.Engine
}

func NewMiner(c *protocol.Chain, accountManager *account.Manager, txPool *protocol.TxPool, newBlockCh chan *bc.Hash, engine engine.Engine) *Miner {
	authoritys := make(map[string]string)
	var position uint64
	dpos, ok := engine.(*dpos.Dpos)
	if !ok {
		log.Error("Only the dpos engine was allowed")
		return nil
	}
	dpos.Authorize(config.CommonConfig.Consensus.Dpos.Coinbase)
	/*
		for index, xpub := range consensus.ActiveNetParams.SignBlockXPubs {
			pubHash := crypto.Ripemd160(xpub.PublicKey())
			address, _ := common.NewPeginAddressWitnessScriptHash(pubHash, &consensus.ActiveNetParams)
			control, _ := vmutil.P2WPKHProgram([]byte(pubHash))
			//key := hex.EncodeToString(control)
			//authoritys[key] = xpub.String()
			authoritys[address.EncodeAddress()] = xpub.String()
			if accountManager.IsLocalControlProgram(control) {
				position = uint64(index)
				dpos.Authorize(address.EncodeAddress())
			}
		}
	*/
	//c.SetAuthoritys(authoritys)
	//c.SetPosition(position)
	c.SetConsensusEngine(dpos)
	ConsensusEngine = dpos
	return &Miner{
		chain:            c,
		accountManager:   accountManager,
		txPool:           txPool,
		numWorkers:       defaultNumWorkers,
		updateNumWorkers: make(chan struct{}),
		newBlockCh:       newBlockCh,
		Authoritys:       authoritys,
		position:         position,
		engine:           dpos,
	}
}

func (m *Miner) generateProof(block types.Block) (types.Proof, error) {
	var xPrv chainkd.XPrv
	if consensus.ActiveNetParams.Signer == "" {
		return types.Proof{}, errors.New("Signer is empty")
	}
	xPrv.UnmarshalText([]byte(consensus.ActiveNetParams.Signer))
	sign := xPrv.Sign(block.BlockCommitment.TransactionsMerkleRoot.Bytes())
	pubHash := crypto.Ripemd160(xPrv.XPub().PublicKey())

	address, _ := common.NewPeginAddressWitnessScriptHash(pubHash, &consensus.ActiveNetParams)
	control, err := vmutil.P2WPKHProgram([]byte(pubHash))
	if err != nil {
		return types.Proof{}, err
	}
	return types.Proof{Sign: sign, ControlProgram: control, Address: address.ScriptAddress()}, nil
}

// generateBlocks is a worker that is controlled by the miningWorkerController.
// It is self contained in that it creates block templates and attempts to solve
// them while detecting when it is performing stale work and reacting
// accordingly by generating a new block template.  When a block is solved, it
// is submitted.
//
// It must be run as a goroutine.
func (m *Miner) generateBlocks(quit chan struct{}) {
	ticker := time.NewTicker(time.Second * hashUpdateSecs)
	defer ticker.Stop()

out:
	for {
		select {
		case <-quit:
			break out
		default:
		}
		/*
			engine, ok := m.engine.(*dpos.Dpos)
			if !ok {
				log.Error("Only the dpos engine was allowed")
				return
			}

				header := m.chain.BestBlockHeader()
				isSeal, err := engine.IsSealer(m.chain, header.Hash(), header, uint64(time.Now().Unix()))
				if err != nil {
					log.WithFields(log.Fields{"module": module, "error": err}).Error("Determine whether seal is wrong")
					continue
				}
		*/
		isSeal := true
		if isSeal {
			block, err := mining.NewBlockTemplate1(m.chain, m.txPool, m.accountManager, m.engine)
			if err != nil {
				log.Errorf("Mining: failed on create NewBlockTemplate: %v", err)
				time.Sleep(3 * time.Second)
				continue
			}
			if block == nil {
				time.Sleep(3 * time.Second)
				continue
			}
			block, err = m.engine.Seal(m.chain, block)
			if err != nil {
				log.Errorf("Seal, %v", err)
				continue
			}
			m.chain.SetConsensusEngine(m.engine)
			if isOrphan, err := m.chain.ProcessBlock(block); err == nil {
				log.WithFields(log.Fields{
					"height":   block.BlockHeader.Height,
					"isOrphan": isOrphan,
					"tx":       len(block.Transactions),
				}).Info("Miner processed block")

				blockHash := block.Hash()
				m.newBlockCh <- &blockHash
			} else {
				log.WithField("height", block.BlockHeader.Height).Errorf("Miner fail on ProcessBlock, %v", err)
			}
		}
		time.Sleep(3 * time.Second)
	}

	m.workerWg.Done()
}

// miningWorkerController launches the worker goroutines that are used to
// generate block templates and solve them.  It also provides the ability to
// dynamically adjust the number of running worker goroutines.
//
// It must be run as a goroutine.
func (m *Miner) miningWorkerController() {
	// launchWorkers groups common code to launch a specified number of
	// workers for generating blocks.
	var runningWorkers []chan struct{}
	launchWorkers := func(numWorkers uint64) {
		for i := uint64(0); i < numWorkers; i++ {
			quit := make(chan struct{})
			runningWorkers = append(runningWorkers, quit)

			m.workerWg.Add(1)
			go m.generateBlocks(quit)
		}
	}

	// Launch the current number of workers by default.
	runningWorkers = make([]chan struct{}, 0, m.numWorkers)
	launchWorkers(m.numWorkers)

out:
	for {
		select {
		// Update the number of running workers.
		case <-m.updateNumWorkers:
			// No change.
			numRunning := uint64(len(runningWorkers))
			if m.numWorkers == numRunning {
				continue
			}

			// Add new workers.
			if m.numWorkers > numRunning {
				launchWorkers(m.numWorkers - numRunning)
				continue
			}

			// Signal the most recently created goroutines to exit.
			for i := numRunning - 1; i >= m.numWorkers; i-- {
				close(runningWorkers[i])
				runningWorkers[i] = nil
				runningWorkers = runningWorkers[:i]
			}

		case <-m.quit:
			for _, quit := range runningWorkers {
				close(quit)
			}
			break out
		}
	}

	m.workerWg.Wait()
}

// Start begins the CPU mining process as well as the speed monitor used to
// track hashing metrics.  Calling this function when the CPU miner has
// already been started will have no effect.
//
// This function is safe for concurrent access.
func (m *Miner) Start() {
	m.Lock()
	defer m.Unlock()

	// Nothing to do if the miner is already running
	if m.started {
		return
	}

	m.quit = make(chan struct{})
	go m.miningWorkerController()

	m.started = true
	log.Infof("CPU miner started")
}

// Stop gracefully stops the mining process by signalling all workers, and the
// speed monitor to quit.  Calling this function when the CPU miner has not
// already been started will have no effect.
//
// This function is safe for concurrent access.
func (m *Miner) Stop() {
	m.Lock()
	defer m.Unlock()

	// Nothing to do if the miner is not currently running
	if !m.started {
		return
	}

	close(m.quit)
	m.started = false
	log.Info("CPU miner stopped")
}

// IsMining returns whether or not the CPU miner has been started and is
// therefore currenting mining.
//
// This function is safe for concurrent access.
func (m *Miner) IsMining() bool {
	m.Lock()
	defer m.Unlock()

	return m.started
}

// SetNumWorkers sets the number of workers to create which solve blocks.  Any
// negative values will cause a default number of workers to be used which is
// based on the number of processor cores in the system.  A value of 0 will
// cause all CPU mining to be stopped.
//
// This function is safe for concurrent access.
func (m *Miner) SetNumWorkers(numWorkers int32) {
	if numWorkers == 0 {
		m.Stop()
	}

	// Don't lock until after the first check since Stop does its own
	// locking.
	m.Lock()
	defer m.Unlock()

	// Use default if provided value is negative.
	if numWorkers < 0 {
		m.numWorkers = defaultNumWorkers
	} else {
		m.numWorkers = uint64(numWorkers)
	}

	// When the miner is already running, notify the controller about the
	// the change.
	if m.started {
		m.updateNumWorkers <- struct{}{}
	}
}

// NumWorkers returns the number of workers which are running to solve blocks.
//
// This function is safe for concurrent access.
func (m *Miner) NumWorkers() int32 {
	m.Lock()
	defer m.Unlock()

	return int32(m.numWorkers)
}
