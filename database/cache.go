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
)

type fillBlockHeaderFn func(hash *bc.Hash, height uint64) (*types.BlockHeader, error)
type fillBlockTransactionsFn func(hash *bc.Hash) ([]*types.Tx, error)
type fillVoteResultFn func(seq uint64) (*state.VoteResult, error)

func newCache(fillBlockHeader fillBlockHeaderFn, fillBlockTxs fillBlockTransactionsFn, fillVoteResult fillVoteResultFn) cache {
	return cache{
		lruBlockHeaders: common.NewCache(maxCachedBlockHeaders),
		lruBlockTxs:     common.NewCache(maxCachedBlockTransactions),
		lruVoteResults:  common.NewCache(maxCachedVoteResults),

		fillBlockHeaderFn:      fillBlockHeader,
		fillBlockTransactionFn: fillBlockTxs,
		fillVoteResultFn:       fillVoteResult,
	}
}

type cache struct {
	lruBlockHeaders *common.Cache
	lruBlockTxs     *common.Cache
	lruVoteResults  *common.Cache

	fillBlockHeaderFn      func(hash *bc.Hash, height uint64) (*types.BlockHeader, error)
	fillBlockTransactionFn func(hash *bc.Hash) ([]*types.Tx, error)
	fillVoteResultFn       func(seq uint64) (*state.VoteResult, error)

	sf singleflight.Group
}

func (c *cache) lookupBlockHeader(hash *bc.Hash, height uint64) (*types.BlockHeader, error) {
	if data, ok := c.lruBlockHeaders.Get(*hash); ok {
		return data.(*types.BlockHeader), nil
	}

	blockHeader, err := c.sf.Do("BlockHeader:"+hash.String(), func() (interface{}, error) {
		blockHeader, err := c.fillBlockHeaderFn(hash, height)
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

func (c *cache) removeBlockHeader(blockHeader *types.BlockHeader) {
	c.lruBlockHeaders.Remove(blockHeader.Hash())
}

func (c *cache) removeVoteResult(voteResult *state.VoteResult) {
	c.lruVoteResults.Remove(voteResult.Seq)
}
