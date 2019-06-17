package database

import (
	"testing"

	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
)

func TestBlockCache(t *testing.T) {
	newBlock := func(h uint64) *types.Block {
		return &types.Block{
			BlockHeader: types.BlockHeader{
				Height: h,
			},
		}
	}
	newVoteResult := func(seq uint64) *state.VoteResult {
		return &state.VoteResult{
			Seq: seq,
		}
	}
	blocks := make(map[bc.Hash]*types.Block)
	for i := 0; i < maxCachedBlockHeaders+10; i++ {
		block := newBlock(uint64(i))
		blocks[block.Hash()] = block
	}
	voteResults := make(map[uint64]*state.VoteResult)
	for i := 0; i < maxCachedVoteResults+10; i++ {
		voteResult := newVoteResult(uint64(i))
		voteResults[voteResult.Seq] = voteResult
	}

	fillBlockHeaderFn := func(hash *bc.Hash, height uint64) (*types.BlockHeader, error) {
		return &blocks[*hash].BlockHeader, nil
	}

	fillBlockTxsFn := func(hash *bc.Hash) ([]*types.Tx, error) {
		return blocks[*hash].Transactions, nil
	}

	fillVoteResultFn := func(seq uint64) (*state.VoteResult, error) {
		return voteResults[seq], nil
	}

	cache := newCache(fillBlockHeaderFn, fillBlockTxsFn, fillVoteResultFn)

	for i := 0; i < maxCachedBlockHeaders+10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		cache.lookupBlockHeader(&hash, block.Height)
	}

	for i := 0; i < 10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		if _, ok := cache.lruBlockHeaders.Get(hash); ok {
			t.Fatalf("find old block")
		}
	}

	for i := 10; i < maxCachedBlockHeaders+10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		if _, ok := cache.lruBlockHeaders.Get(hash); !ok {
			t.Fatalf("can't find new block")
		}
	}

	for i := 0; i < maxCachedVoteResults+10; i++ {
		voteResult := newVoteResult(uint64(i))
		cache.lookupVoteResult(voteResult.Seq)
	}

	for i := 0; i < 10; i++ {
		voteResult := newVoteResult(uint64(i))
		if _, ok := cache.lruVoteResults.Get(voteResult.Seq); ok {
			t.Fatalf("find old vote result")
		}
	}

	for i := 10; i < maxCachedVoteResults+10; i++ {
		voteResult := newVoteResult(uint64(i))
		if _, ok := cache.lruVoteResults.Get(voteResult.Seq); !ok {
			t.Fatalf("can't find new vote result")
		}
	}
}
