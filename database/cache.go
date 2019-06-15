package database

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/golang/groupcache/lru"
	"github.com/golang/groupcache/singleflight"

	"github.com/vapor/common"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
)

const (
	maxCachedBlockHeaders      = 1000
	maxCachedBlockTransactions = 1000
	maxCachedVoteResults       = 144 // int(60 * 60 * 24 * 1000 / consensus.BlockTimeInterval / consensus.RoundVoteBlockNums)
)

type fillBlockHeaderFn func(hash *bc.Hash, height uint64) (*types.BlockHeader, error)
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

func newVoteResultCache(fillFn func(seq uint64) (*state.VoteResult, error)) voteResultCache {
	return voteResultCache{
		lru:    lru.New(maxCachedVoteResults),
		fillFn: fillFn,
	}
}

type voteResultCache struct {
	mu     sync.Mutex
	lru    *lru.Cache
	fillFn func(seq uint64) (*state.VoteResult, error)
	single singleflight.Group
}

func (vrc *voteResultCache) lookup(seq uint64) (*state.VoteResult, error) {
	if voteResult, ok := vrc.get(seq); ok {
		return voteResult, nil
	}

	seqStr := strconv.FormatUint(seq, 10)
	voteResult, err := vrc.single.Do(seqStr, func() (interface{}, error) {
		v, err := vrc.fillFn(seq)
		if err != nil {
			return nil, err
		}

		if v == nil {
			return nil, fmt.Errorf("There are no vote result with given seq %s", seqStr)
		}

		vrc.add(v)
		return v, nil
	})
	if err != nil {
		return nil, err
	}
	return voteResult.(*state.VoteResult), nil
}

func (vrc *voteResultCache) get(seq uint64) (*state.VoteResult, bool) {
	vrc.mu.Lock()
	voteResult, ok := vrc.lru.Get(seq)
	vrc.mu.Unlock()
	if voteResult == nil {
		return nil, ok
	}
	return voteResult.(*state.VoteResult), ok
}

func (vrc *voteResultCache) add(voteResult *state.VoteResult) {
	vrc.mu.Lock()
	vrc.lru.Add(voteResult.Seq, voteResult)
	vrc.mu.Unlock()
}
