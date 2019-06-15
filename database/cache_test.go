package database

import (
	"testing"

	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
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
	for i := 0; i < maxCachedBlocks+10; i++ {
		block := newBlock(uint64(i))
		blocks[block.Hash()] = block
	}

	cache := newBlockCache(func(hash *bc.Hash, height uint64) (*types.Block, error) {
		return blocks[*hash], nil
	})

	for i := 0; i < maxCachedBlocks+10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		cache.lookup(&hash, block.Height)
	}

	for i := 0; i < 10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		if b, _ := cache.get(&hash); b != nil {
			t.Fatalf("find old block")
		}
	}

	for i := 10; i < maxCachedBlocks+10; i++ {
		block := newBlock(uint64(i))
		hash := block.Hash()
		if b, _ := cache.get(&hash); b == nil {
			t.Fatalf("can't find new block")
		}
	}
}
