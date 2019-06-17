package database

import (
	"fmt"

	"github.com/golang/groupcache/singleflight"

	"github.com/vapor/common"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	maxCachedBlockHeaders      = 1000
	maxCachedBlockTransactions = 1000
)

type fillBlockHeaderFn func(hash *bc.Hash) (*types.BlockHeader, error)
type fillBlockTransactionsFn func(hash *bc.Hash) ([]*types.Tx, error)

func newBlockCache(fillBlockHeader fillBlockHeaderFn, fillBlockTxs fillBlockTransactionsFn) blockCache {
	return blockCache{
		lruBlockHeaders: common.NewCache(maxCachedBlockHeaders),
		lruBlockTxs:     common.NewCache(maxCachedBlockTransactions),

		fillBlockHeaderFn:      fillBlockHeader,
		fillBlockTransactionFn: fillBlockTxs,
	}
}

type blockCache struct {
	lruBlockHeaders *common.Cache
	lruBlockTxs     *common.Cache

	fillBlockHeaderFn      func(hash *bc.Hash) (*types.BlockHeader, error)
	fillBlockTransactionFn func(hash *bc.Hash) ([]*types.Tx, error)

	singleBlockHeader singleflight.Group
	singleBlockTxs    singleflight.Group
}

func (c *blockCache) lookupBlockHeader(hash *bc.Hash) (*types.BlockHeader, error) {
	if bH, ok := c.getBlockHeader(hash); ok {
		return bH, nil
	}

	blockHeader, err := c.singleBlockHeader.Do(hash.String(), func() (interface{}, error) {
		bH, err := c.fillBlockHeaderFn(hash)
		if err != nil {
			return nil, err
		}

		if bH == nil {
			return nil, fmt.Errorf("There are no blockHeader with given hash %s", hash.String())
		}

		c.addBlockHeader(bH)
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

		c.addBlockTxs(*hash, bTxs)
		return bTxs, nil
	})
	if err != nil {
		return nil, err
	}
	return blockTransactions.([]*types.Tx), nil
}

func (c *blockCache) getBlockHeader(hash *bc.Hash) (*types.BlockHeader, bool) {
	blockHeader, ok := c.lruBlockHeaders.Get(*hash)
	if blockHeader == nil {
		return nil, ok
	}
	return blockHeader.(*types.BlockHeader), ok
}

func (c *blockCache) getBlockTransactions(hash *bc.Hash) ([]*types.Tx, bool) {
	txs, ok := c.lruBlockTxs.Get(*hash)
	if txs == nil {
		return nil, ok
	}
	return txs.([]*types.Tx), ok
}

func (c *blockCache) addBlockHeader(blockHeader *types.BlockHeader) {
	c.lruBlockHeaders.Add(blockHeader.Hash(), blockHeader)
}

func (c *blockCache) addBlockTxs(hash bc.Hash, txs []*types.Tx) {
	c.lruBlockTxs.Add(hash, txs)
}
