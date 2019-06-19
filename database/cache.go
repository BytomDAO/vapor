package database

import (
	"strconv"

	"github.com/golang/groupcache/singleflight"

	"github.com/vapor/common"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
)

const (
	maxCachedBlockHeaders      = 4096
	maxCachedBlockTransactions = 1024
	maxCachedVoteResults       = 128
	maxCachedHeightIndexes     = 8192
	maxCachedMainChainHashes   = 8192
)

type fillBlockHeaderFn func(hash *bc.Hash) (*types.BlockHeader, error)
type fillBlockTransactionsFn func(hash *bc.Hash) ([]*types.Tx, error)
type fillVoteResultFn func(seq uint64) (*state.VoteResult, error)
type fillHeightIndexFn func(height uint64) ([]*bc.Hash, error)
type fillMainChainHashFn func(height uint64) (*bc.Hash, error)

func newCache(fillBlockHeader fillBlockHeaderFn, fillBlockTxs fillBlockTransactionsFn, fillVoteResult fillVoteResultFn,
	fillHeightIndex fillHeightIndexFn, fillMainChainHash fillMainChainHashFn) cache {
	return cache{
		lruBlockHeaders:    common.NewCache(maxCachedBlockHeaders),
		lruBlockTxs:        common.NewCache(maxCachedBlockTransactions),
		lruVoteResults:     common.NewCache(maxCachedVoteResults),
		lruHeightIndexes:   common.NewCache(maxCachedHeightIndexes),
		lruMainChainHashes: common.NewCache(maxCachedMainChainHashes),

		fillBlockHeaderFn:      fillBlockHeader,
		fillBlockTransactionFn: fillBlockTxs,
		fillVoteResultFn:       fillVoteResult,
		fillHeightIndexFn:      fillHeightIndex,
		fillMainChainHashFn:    fillMainChainHash,
	}
}

type cache struct {
	lruBlockHeaders    *common.Cache
	lruBlockTxs        *common.Cache
	lruVoteResults     *common.Cache
	lruHeightIndexes   *common.Cache
	lruMainChainHashes *common.Cache

	fillBlockHeaderFn      func(hash *bc.Hash) (*types.BlockHeader, error)
	fillBlockTransactionFn func(hash *bc.Hash) ([]*types.Tx, error)
	fillVoteResultFn       func(seq uint64) (*state.VoteResult, error)
	fillHeightIndexFn      func(uint64) ([]*bc.Hash, error)
	fillMainChainHashFn    func(uint64) (*bc.Hash, error)

	sf singleflight.Group
}

func (c *cache) lookupBlockHeader(hash *bc.Hash) (*types.BlockHeader, error) {
	if data, ok := c.lruBlockHeaders.Get(*hash); ok {
		return data.(*types.BlockHeader), nil
	}

	blockHeader, err := c.sf.Do("BlockHeader:"+hash.String(), func() (interface{}, error) {
		blockHeader, err := c.fillBlockHeaderFn(hash)
		if err != nil {
			return nil, err
		}

		c.lruBlockHeaders.Add(blockHeader.Hash(), blockHeader)
		return blockHeader, nil
	})
	if err != nil {
		return nil, err
	}
	return blockHeader.(*types.BlockHeader), nil
}

func (c *cache) lookupBlockTxs(hash *bc.Hash) ([]*types.Tx, error) {
	if data, ok := c.lruBlockTxs.Get(*hash); ok {
		return data.([]*types.Tx), nil
	}

	blockTxs, err := c.sf.Do("BlockTxs:"+hash.String(), func() (interface{}, error) {
		blockTxs, err := c.fillBlockTransactionFn(hash)
		if err != nil {
			return nil, err
		}

		c.lruBlockTxs.Add(hash, blockTxs)
		return blockTxs, nil
	})
	if err != nil {
		return nil, err
	}
	return blockTxs.([]*types.Tx), nil
}

func (c *cache) lookupVoteResult(seq uint64) (*state.VoteResult, error) {
	if data, ok := c.lruVoteResults.Get(seq); ok {
		return data.(*state.VoteResult).Fork(), nil
	}

	seqStr := strconv.FormatUint(seq, 10)
	voteResult, err := c.sf.Do("VoteResult:"+seqStr, func() (interface{}, error) {
		voteResult, err := c.fillVoteResultFn(seq)
		if err != nil {
			return nil, err
		}

		c.lruVoteResults.Add(voteResult.Seq, voteResult)
		return voteResult, nil
	})
	if err != nil {
		return nil, err
	}
	return voteResult.(*state.VoteResult).Fork(), nil
}

func (c *cache) lookupMainChainHash(height uint64) (*bc.Hash, error) {
	if hash, ok := c.lruMainChainHashes.Get(height); ok {
		return hash.(*bc.Hash), nil
	}

	heightStr := strconv.FormatUint(height, 10)
	hash, err := c.sf.Do("BlockHashByHeight:"+heightStr, func() (interface{}, error) {
		hash, err := c.fillMainChainHashFn(height)
		if err != nil {
			return nil, err
		}

		c.lruMainChainHashes.Add(height, hash)
		return hash, nil
	})
	if err != nil {
		return nil, err
	}
	return hash.(*bc.Hash), nil
}

func (c *cache) lookupBlockHashesByHeight(height uint64) ([]*bc.Hash, error) {
	if hashes, ok := c.lruHeightIndexes.Get(height); ok {
		return hashes.([]*bc.Hash), nil
	}

	heightStr := strconv.FormatUint(height, 10)
	hashes, err := c.sf.Do("BlockHashesByHeight:"+heightStr, func() (interface{}, error) {
		hashes, err := c.fillHeightIndexFn(height)
		if err != nil {
			return nil, err
		}

		c.lruHeightIndexes.Add(height, hashes)
		return hashes, nil
	})
	if err != nil {
		return nil, err
	}
	return hashes.([]*bc.Hash), nil
}

func (c *cache) removeBlockHeader(blockHeader *types.BlockHeader) {
	c.lruBlockHeaders.Remove(blockHeader.Hash())
}

func (c *cache) removeVoteResult(voteResult *state.VoteResult) {
	c.lruVoteResults.Remove(voteResult.Seq)
}
