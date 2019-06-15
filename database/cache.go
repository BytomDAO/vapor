package database

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/golang/groupcache/lru"
	"github.com/golang/groupcache/singleflight"

	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
)

const (
	maxCachedBlocks      = 30
	maxCachedVoteResults = 30
)

func newBlockCache(fillFn func(hash *bc.Hash) (*types.Block, error)) blockCache {
	return blockCache{
		lru:    lru.New(maxCachedBlocks),
		fillFn: fillFn,
	}
}

type blockCache struct {
	mu     sync.Mutex
	lru    *lru.Cache
	fillFn func(hash *bc.Hash) (*types.Block, error)
	single singleflight.Group
}

func (c *blockCache) lookup(hash *bc.Hash) (*types.Block, error) {
	if b, ok := c.get(hash); ok {
		return b, nil
	}

	block, err := c.single.Do(hash.String(), func() (interface{}, error) {
		b, err := c.fillFn(hash)
		if err != nil {
			return nil, err
		}

		if b == nil {
			return nil, fmt.Errorf("There are no block with given hash %s", hash.String())
		}

		c.add(b)
		return b, nil
	})
	if err != nil {
		return nil, err
	}
	return block.(*types.Block), nil
}

func (c *blockCache) get(hash *bc.Hash) (*types.Block, bool) {
	c.mu.Lock()
	block, ok := c.lru.Get(*hash)
	c.mu.Unlock()
	if block == nil {
		return nil, ok
	}
	return block.(*types.Block), ok
}

func (c *blockCache) add(block *types.Block) {
	c.mu.Lock()
	c.lru.Add(block.Hash(), block)
	c.mu.Unlock()
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
