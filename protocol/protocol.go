package protocol

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/vapor/common"
	"github.com/bytom/vapor/config"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/event"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
)

const (
	maxProcessBlockChSize = 1024
	maxKnownTxs           = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)
)

// Protocoler is interface for layer 2 consensus protocol
type Protocoler interface {
	Name() string
	StartHeight() uint64
	BeforeProposalBlock(txs []*types.Tx, blockHeight uint64, gasLeft int64, isTimeout func() bool) ([]*types.Tx, error)
	ChainStatus() (uint64, *bc.Hash, error)
	ValidateBlock(block *types.Block, verifyResults []*bc.TxVerifyResult) error
	ValidateTx(tx *types.Tx, verifyResult *bc.TxVerifyResult, blockHeight uint64) error
	ApplyBlock(block *types.Block) error
	DetachBlock(block *types.Block) error
}

// Chain provides functions for working with the Bytom block chain.
type Chain struct {
	orphanManage   *OrphanManage
	txPool         *TxPool
	store          Store
	processBlockCh chan *processBlockMsg
	subProtocols   []Protocoler

	signatureCache  *common.Cache
	eventDispatcher *event.Dispatcher

	cond               sync.Cond
	bestBlockHeader    *types.BlockHeader // the last block on current main chain
	lastIrrBlockHeader *types.BlockHeader // the last irreversible block

	knownTxs *common.OrderedSet
}

// NewChain returns a new Chain using store as the underlying storage.
func NewChain(store Store, txPool *TxPool, subProtocols []Protocoler, eventDispatcher *event.Dispatcher) (*Chain, error) {
	knownTxs, _ := common.NewOrderedSet(maxKnownTxs)
	c := &Chain{
		orphanManage:    NewOrphanManage(),
		txPool:          txPool,
		store:           store,
		subProtocols:    subProtocols,
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

	for _, p := range c.subProtocols {
		if err := c.syncProtocolStatus(p); err != nil {
			return nil, errors.Wrap(err, p.Name(), "sync sub protocol status")
		}
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

	for _, subProtocol := range c.subProtocols {
		if err := subProtocol.ApplyBlock(genesisBlock); err != nil {
			return err
		}
	}

	consensusResults := []*state.ConsensusResult{{
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

// SubProtocols return list of layer 2 consensus protocol
func (c *Chain) SubProtocols() []Protocoler {
	return c.subProtocols
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

func (c *Chain) syncProtocolStatus(subProtocol Protocoler) error {
	if c.bestBlockHeader.Height < subProtocol.StartHeight() {
		return nil
	}

	protocolHeight, protocolHash, err := subProtocol.ChainStatus()
	if err != nil {
		return errors.Wrap(err, "failed on get sub protocol status")
	}

	if *protocolHash == c.bestBlockHeader.Hash() {
		return nil
	}

	for !c.InMainChain(*protocolHash) {
		block, err := c.GetBlockByHash(protocolHash)
		if err != nil {
			return errors.Wrap(err, subProtocol.Name(), "can't get block by hash in chain")
		}

		if err := subProtocol.DetachBlock(block); err != nil {
			return errors.Wrap(err, subProtocol.Name(), "sub protocol detach block err")
		}

		protocolHeight, protocolHash = block.Height-1, &block.PreviousBlockHash
	}

	for height := protocolHeight + 1; height <= c.bestBlockHeader.Height; height++ {
		block, err := c.GetBlockByHeight(height)
		if err != nil {
			return errors.Wrap(err, subProtocol.Name(), "can't get block by height in chain")
		}

		if err := subProtocol.ApplyBlock(block); err != nil {
			return errors.Wrap(err, subProtocol.Name(), "sub protocol apply block err")
		}

		blockHash := block.Hash()
		protocolHeight, protocolHash = block.Height, &blockHash
	}

	return nil
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
