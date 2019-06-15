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
	blocks := make(map[bc.Hash]*types.Block)
	for i := 0; i < maxCachedBlockHeaders+10; i++ {
		block := newBlock(uint64(i))
		blocks[block.Hash()] = block
	}

	fillBlockHeaderFn := func(hash *bc.Hash, height uint64) (*types.BlockHeader, error) {
		return &blocks[*hash].BlockHeader, nil
	}

	fillBlockTxsFn := func(hash *bc.Hash) ([]*types.Tx, error) {
		return blocks[*hash].Transactions, nil
	}

	cache := newBlockCache(fillBlockHeaderFn, fillBlockTxsFn)

	for i := 0; i < maxCachedBlockHeaders+10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		cache.lookupBlockHeader(&hash, block.Height)
	}

	for i := 0; i < 10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		if b, _ := cache.getBlockHeader(&hash); b != nil {
			t.Fatalf("find old block")
		}
	}

	for i := 10; i < maxCachedBlockHeaders+10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		if b, _ := cache.getBlockHeader(&hash); b == nil {
			t.Fatalf("can't find new block")
		}
	}
}

func TestVoteResultCache(t *testing.T) {
	newVoteResult := func(seq uint64) *state.VoteResult {
		return &state.VoteResult{
			Seq: seq,
		}
	}
	voteResults := make(map[uint64]*state.VoteResult)
	for i := 0; i < maxCachedVoteResults+10; i++ {
		voteResult := newVoteResult(uint64(i))
		voteResults[voteResult.Seq] = voteResult
	}

	cache := newVoteResultCache(func(seq uint64) (*state.VoteResult, error) {
		return voteResults[seq], nil
	})

	for i := 0; i < maxCachedVoteResults+10; i++ {
		voteResult := newVoteResult(uint64(i))
		cache.lookup(voteResult.Seq)
	}

	for i := 0; i < 10; i++ {
		voteResult := newVoteResult(uint64(i))
		if v, _ := cache.get(voteResult.Seq); v != nil {
			t.Fatalf("find old vote result")
		}
	}

	for i := 10; i < maxCachedVoteResults+10; i++ {
		voteResult := newVoteResult(uint64(i))
		if v, _ := cache.get(voteResult.Seq); v == nil {
			t.Fatalf("can't find new vote result")
		}
	}
}
