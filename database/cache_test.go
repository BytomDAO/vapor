package database

import (
	"testing"

	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
)

func TestBlockCache(t *testing.T) {
	newBlock := func(h uint64) *types.Block {
		return &types.Block{
			BlockHeader: types.BlockHeader{
				Height: h,
			},
		}
	}

	newConsensusResult := func(seq uint64) *state.ConsensusResult {
		return &state.ConsensusResult{
			Seq: seq,
		}
	}

	blocks := make(map[bc.Hash]*types.Block)
	blockHashes := make(map[uint64]*bc.Hash)
	blockIndexHashes := make(map[uint64][]*bc.Hash)
	for i := 0; i < maxCachedBlockHeaders+10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		blocks[hash] = block
		blockHashes[block.Height] = &hash
		blockIndexHashes[block.Height] = append(blockIndexHashes[block.Height], &hash)
	}

	consensusResults := make(map[uint64]*state.ConsensusResult)
	for i := 0; i < maxCachedConsensusResults+10; i++ {
		consensusResult := newConsensusResult(uint64(i))
		consensusResults[consensusResult.Seq] = consensusResult
	}

	fillBlockHeaderFn := func(hash *bc.Hash) (*types.BlockHeader, error) {
		return &blocks[*hash].BlockHeader, nil
	}

	fillBlockTxsFn := func(hash *bc.Hash) ([]*types.Tx, error) {
		return blocks[*hash].Transactions, nil
	}

	fillBlockHashesFn := func(height uint64) ([]*bc.Hash, error) {
		return blockIndexHashes[height], nil
	}

	fillMainChainHashFn := func(height uint64) (*bc.Hash, error) {
		return blockHashes[height], nil
	}

	fillConsensusResultFn := func(seq uint64) (*state.ConsensusResult, error) {
		return consensusResults[seq], nil
	}

	cache := newCache(fillBlockHeaderFn, fillBlockTxsFn, fillBlockHashesFn, fillMainChainHashFn, fillConsensusResultFn)
	for i := 0; i < maxCachedBlockHeaders+10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		cache.lookupBlockHeader(&hash)
	}

	for i := 0; i < 10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		if _, ok := cache.lruBlockHeaders.Get(hash); ok {
			t.Fatalf("find old block header")
		}
	}

	for i := 10; i < maxCachedBlockHeaders+10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		if _, ok := cache.lruBlockHeaders.Get(hash); !ok {
			t.Fatalf("can't find new block header")
		}
	}

	for i := 0; i < maxCachedBlockTransactions+10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		cache.lookupBlockTxs(&hash)
	}

	for i := 0; i < 10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		if _, ok := cache.lruBlockTxs.Get(hash); ok {
			t.Fatalf("find old block transactions")
		}
	}

	for i := 10; i < maxCachedBlockTransactions+10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		if _, ok := cache.lruBlockTxs.Get(hash); !ok {
			t.Fatalf("can't find new block transactions")
		}
	}

	for i := 0; i < maxCachedBlockHashes+10; i++ {
		block := newBlock(uint64(i))
		cache.lookupBlockHashesByHeight(block.Height)
	}

	for i := 0; i < 10; i++ {
		block := newBlock(uint64(i))
		if _, ok := cache.lruBlockHashes.Get(block.Height); ok {
			t.Fatalf("find old block Hashes for specified height")
		}
	}

	for i := 10; i < maxCachedBlockHashes+10; i++ {
		block := newBlock(uint64(i))
		if _, ok := cache.lruBlockHashes.Get(block.Height); !ok {
			t.Fatalf("can't find new block Hashes for specified height")
		}
	}

	for i := 0; i < maxCachedMainChainHashes+10; i++ {
		block := newBlock(uint64(i))
		cache.lookupMainChainHash(block.Height)
	}

	for i := 0; i < 10; i++ {
		block := newBlock(uint64(i))
		if _, ok := cache.lruMainChainHashes.Get(block.Height); ok {
			t.Fatalf("find old main chain block Hash for specified height")
		}
	}

	for i := 10; i < maxCachedMainChainHashes+10; i++ {
		block := newBlock(uint64(i))
		if _, ok := cache.lruMainChainHashes.Get(block.Height); !ok {
			t.Fatalf("can't find new main chain block Hash for specified height")
		}
	}

	for i := 0; i < maxCachedConsensusResults+10; i++ {
		consensusResult := newConsensusResult(uint64(i))
		cache.lookupConsensusResult(consensusResult.Seq)
	}

	for i := 0; i < 10; i++ {
		consensusResult := newConsensusResult(uint64(i))
		if _, ok := cache.lruConsensusResults.Get(consensusResult.Seq); ok {
			t.Fatalf("find old vote result")
		}
	}

	for i := 10; i < maxCachedConsensusResults+10; i++ {
		consensusResult := newConsensusResult(uint64(i))
		if _, ok := cache.lruConsensusResults.Get(consensusResult.Seq); !ok {
			t.Fatalf("can't find new vote result")
		}
	}
}
