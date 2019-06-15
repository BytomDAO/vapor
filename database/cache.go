package database

import (
	"fmt"
	"sync"

	"github.com/golang/groupcache/lru"
	"github.com/golang/groupcache/singleflight"

	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	maxCachedBlockHeaders     = 1000
	maxCachedBlockTransactons = 1000
)

type fillBlockHeaderFn func(hash *bc.Hash, height uint64) (*types.BlockHeader, error)
type fillBlockTransactionsFn func(hash *bc.Hash) ([]*types.Tx, error)

func newBlockCache(fillBlockHeader fillBlockHeaderFn, fillBlockTxs fillBlockTransactionsFn) blockCache {
	return blockCache{
		lruBlockHeaders: lru.New(maxCachedBlockHeaders),
		lruTxs:          lru.New(maxCachedBlockTransactons),

		fillBlockHeaderFn:      fillBlockHeader,
		fillBlockTransactionFn: fillBlockTxs,
	}
}

type blockCache struct {
	muHeaders sync.Mutex
	muTxs     sync.Mutex

	lruBlockHeaders *lru.Cache
	lruTxs          *lru.Cache

	fillBlockHeaderFn      func(hash *bc.Hash, height uint64) (*types.BlockHeader, error)
	fillBlockTransactionFn func(hash *bc.Hash) ([]*types.Tx, error)

	singleBlockHeader singleflight.Group
	singleBlockTxs    singleflight.Group
}

func (c *blockCache) lookupBlockHeader(hash *bc.Hash, height uint64) (*types.BlockHeader, error) {
	if bH, ok := c.getBlockHeader(hash); ok {
		return bH, nil
	}

	blockHeader, err := c.singleBlockHeader.Do(hash.String(), func() (interface{}, error) {
		bH, err := c.fillBlockHeaderFn(hash, height)
		if err != nil {
			return nil, err
		}

		if bH == nil {
			return nil, fmt.Errorf("There are no blockHeader with given hash %s", hash.String())
		}

		c.addHeader(bH)
		return bH, nil
	})
	if err != nil {
		return nil, err
	}
	return blockHeader.(*types.BlockHeader), nil
}

func (c *blockCache) lookupBlockTxs(hash *bc.Hash) ([]*types.Tx, error) {
	if bTxs, ok := c.getBlockTransactions(hash); ok {
		return bTxs, nil
	}

	blockTransactions, err := c.singleBlockTxs.Do(hash.String(), func() (interface{}, error) {
		bTxs, err := c.fillBlockTransactionFn(hash)
		if err != nil {
			return nil, err
		}

		if bTxs == nil {
			return nil, fmt.Errorf("There are no block transactions with given hash %s", hash.String())
		}

		c.addTxs(*hash, bTxs)
		return bTxs, nil
	})
	if err != nil {
		return nil, err
	}
	return blockTransactions.([]*types.Tx), nil
}

func (c *blockCache) getBlockHeader(hash *bc.Hash) (*types.BlockHeader, bool) {
	c.muHeaders.Lock()
	blockHeader, ok := c.lruBlockHeaders.Get(*hash)
	c.muHeaders.Unlock()
	if blockHeader == nil {
		return nil, ok
	}
	return blockHeader.(*types.BlockHeader), ok
}

func (c *blockCache) getBlockTransactions(hash *bc.Hash) ([]*types.Tx, bool) {
	c.muTxs.Lock()
	txs, ok := c.lruTxs.Get(*hash)
	c.muTxs.Unlock()
	if txs == nil {
		return nil, ok
	}
	return txs.([]*types.Tx), ok
}

func (c *blockCache) addHeader(blockHeader *types.BlockHeader) {
	c.muHeaders.Lock()
	c.lruBlockHeaders.Add(blockHeader.Hash(), blockHeader)
	c.muHeaders.Unlock()
}

func (c *blockCache) addTxs(hash bc.Hash, txs []*types.Tx) {
	c.muTxs.Lock()
	c.lruTxs.Add(hash, txs)
	c.muTxs.Unlock()
}
