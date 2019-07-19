package protocol

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/common"
	"github.com/vapor/config"
	"github.com/vapor/event"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
)

const (
	maxProcessBlockChSize = 1024
	maxKnownTxs           = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)
)

// Chain provides functions for working with the Bytom block chain.
type Chain struct {
	orphanManage   *OrphanManage
	txPool         *TxPool
	store          Store
	processBlockCh chan *processBlockMsg

	signatureCache  *common.Cache
	eventDispatcher *event.Dispatcher

	cond               sync.Cond
	bestBlockHeader    *types.BlockHeader // the last block on current main chain
	lastIrrBlockHeader *types.BlockHeader // the last irreversible block

	knownTxs *common.OrderedSet
}

// NewChain returns a new Chain using store as the underlying storage.
func NewChain(store Store, txPool *TxPool, eventDispatcher *event.Dispatcher) (*Chain, error) {
	knownTxs, _ := common.NewOrderedSet(maxKnownTxs)
	c := &Chain{
		orphanManage:    NewOrphanManage(),
		txPool:          txPool,
		store:           store,
		signatureCache:  common.NewCache(maxSignatureCacheSize),
		eventDispatcher: eventDispatcher,
		processBlockCh:  make(chan *processBlockMsg, maxProcessBlockChSize),
		knownTxs:        knownTxs,
	}
	c.cond.L = new(sync.Mutex)

	storeStatus := store.GetStoreStatus()
	if storeStatus == nil {
		if err := c.initChainStatus(); err != nil {
			return nil, err
		}
		storeStatus = store.GetStoreStatus()
	}

	var err error
	c.bestBlockHeader, err = c.store.GetBlockHeader(storeStatus.Hash)
	if err != nil {
		return nil, err
	}

	c.lastIrrBlockHeader, err = c.store.GetBlockHeader(storeStatus.IrreversibleHash)
	if err != nil {
		return nil, err
	}
	go c.blockProcesser()
	return c, nil
}

func (c *Chain) initChainStatus() error {
	genesisBlock := config.GenesisBlock()
	txStatus := bc.NewTransactionStatus()
	for i := range genesisBlock.Transactions {
		if err := txStatus.SetStatus(i, false); err != nil {
			return err
		}
	}

	if err := c.store.SaveBlock(genesisBlock, txStatus); err != nil {
		return err
	}

	utxoView := state.NewUtxoViewpoint()
	bcBlock := types.MapBlock(genesisBlock)
	if err := utxoView.ApplyBlock(bcBlock, txStatus); err != nil {
		return err
	}

	consensusResults := []*state.ConsensusResult{&state.ConsensusResult{
		Seq:            0,
		NumOfVote:      make(map[string]uint64),
		CoinbaseReward: make(map[string]uint64),
		BlockHash:      genesisBlock.Hash(),
		BlockHeight:    0,
	}}

	genesisBlockHeader := &genesisBlock.BlockHeader
	return c.store.SaveChainStatus(genesisBlockHeader, genesisBlockHeader, []*types.BlockHeader{genesisBlockHeader}, utxoView, consensusResults)
}

// BestBlockHeight returns the current height of the blockchain.
func (c *Chain) BestBlockHeight() uint64 {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	return c.bestBlockHeader.Height
}

// BestBlockHash return the hash of the main chain tail block
func (c *Chain) BestBlockHash() *bc.Hash {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	bestHash := c.bestBlockHeader.Hash()
	return &bestHash
}

// LastIrreversibleHeader returns the chain last irreversible block header
func (c *Chain) LastIrreversibleHeader() *types.BlockHeader {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	return c.lastIrrBlockHeader
}

// BestBlockHeader returns the chain best block header
func (c *Chain) BestBlockHeader() *types.BlockHeader {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	return c.bestBlockHeader
}

// InMainChain checks wheather a block is in the main chain
func (c *Chain) InMainChain(hash bc.Hash) bool {
	blockHeader, err := c.store.GetBlockHeader(&hash)
	if err != nil {
		return false
	}

	blockHash, err := c.store.GetMainChainHash(blockHeader.Height)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "height": blockHeader.Height}).Debug("not contain block hash in main chain for specified height")
		return false
	}
	return *blockHash == hash
}

// trace back to the tail of the chain from the given block header
func (c *Chain) traceLongestChainTail(blockHeader *types.BlockHeader) (*types.BlockHeader, error) {
	longestTail, workQueue := blockHeader, []*types.BlockHeader{blockHeader}

	for ; len(workQueue) > 0; workQueue = workQueue[1:] {
		currentHeader := workQueue[0]
		currentHash := currentHeader.Hash()
		hashes, err := c.store.GetBlockHashesByHeight(currentHeader.Height + 1)
		if err != nil {
			return nil, err
		}

		for _, h := range hashes {
			if header, err := c.store.GetBlockHeader(h); err != nil {
				return nil, err
			} else if header.PreviousBlockHash == currentHash {
				if longestTail.Height < header.Height {
					longestTail = header
				}
				workQueue = append(workQueue, header)
			}
		}
	}
	return longestTail, nil
}

func (c *Chain) hasSeenTx(tx *types.Tx) bool {
	return c.knownTxs.Has(tx.ID.String())
}

func (c *Chain) markTransactions(txs ...*types.Tx) {
	for _, tx := range txs {
		c.knownTxs.Add(tx.ID.String())
	}
}

// This function must be called with mu lock in above level
func (c *Chain) setState(blockHeader, irrBlockHeader *types.BlockHeader, mainBlockHeaders []*types.BlockHeader, view *state.UtxoViewpoint, consensusResults []*state.ConsensusResult) error {
	if err := c.store.SaveChainStatus(blockHeader, irrBlockHeader, mainBlockHeaders, view, consensusResults); err != nil {
		return err
	}

	c.bestBlockHeader = blockHeader
	c.lastIrrBlockHeader = irrBlockHeader

	blockHash := blockHeader.Hash()
	log.WithFields(log.Fields{"module": logModule, "height": blockHeader.Height, "hash": blockHash.String()}).Debug("chain best status has been update")
	c.cond.Broadcast()
	return nil
}

// BlockWaiter returns a channel that waits for the block at the given height.
func (c *Chain) BlockWaiter(height uint64) <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		c.cond.L.Lock()
		defer c.cond.L.Unlock()
		for c.bestBlockHeader.Height < height {
			c.cond.Wait()
		}
		ch <- struct{}{}
	}()

	return ch
}

// GetTxPool return chain txpool.
func (c *Chain) GetTxPool() *TxPool {
	return c.txPool
}
