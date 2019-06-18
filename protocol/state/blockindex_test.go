package state

import (
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/vapor/protocol/bc"
)

func TestSetMainChain(t *testing.T) {
	blockIndex := NewBlockIndex()
	var lastNode *BlockNode
	for i := uint64(0); i < 4; i++ {
		node := &BlockNode{
			Height: i,
			Hash:   bc.Hash{V0: i},
			Parent: lastNode,
		}
		blockIndex.AddNode(node)
		lastNode = node
	}

	tailNode := lastNode
	blockIndex.SetMainChain(lastNode)
	for lastNode.Parent != nil {
		if !blockIndex.InMainchain(lastNode.Hash) {
			t.Fatalf("block %d, hash %v is not in main chain", lastNode.Height, lastNode.Hash)
		}
		lastNode = lastNode.Parent
	}

	// fork and set main chain
	forkHeight := uint64(1)
	lastNode = blockIndex.nodeByHeight(forkHeight)
	for i := uint64(1); i <= 3; i++ {
		node := &BlockNode{
			Height: lastNode.Height + 1,
			Hash:   bc.Hash{V1: uint64(i)},
			Parent: lastNode,
		}
		blockIndex.AddNode(node)
		lastNode = node
	}

	bestNode := lastNode
	blockIndex.SetMainChain(lastNode)
	for lastNode.Parent != nil {
		if !blockIndex.InMainchain(lastNode.Hash) {
			t.Fatalf("after fork, block %d, hash %v is not in main chain", lastNode.Height, lastNode.Hash)
		}
		lastNode = lastNode.Parent
	}

	if bestNode != blockIndex.BestNode() {
		t.Fatalf("check best node failed")
	}

	for tailNode.Parent != nil && tailNode.Height > forkHeight {
		if blockIndex.InMainchain(tailNode.Hash) {
			t.Fatalf("old chain block %d, hash %v still in main chain", tailNode.Height, tailNode.Hash)
		}
		tailNode = tailNode.Parent
	}
}

// MockBlockIndex will mock a empty BlockIndex
func MockBlockIndex() *BlockIndex {
	return &BlockIndex{
		index:       make(map[bc.Hash]*BlockNode),
		heightIndex: make(map[uint64][]*BlockNode),
		mainChain:   make([]*BlockNode, 0, 2),
	}
}

func TestSetMainChainExtendCap(t *testing.T) {
	blockIndex := MockBlockIndex()
	var lastNode *BlockNode

	cases := []struct {
		start   uint64
		stop    uint64
		wantLen int
		wantCap int
	}{
		{
			start:   0,
			stop:    500,
			wantLen: 500,
			wantCap: 500 + approxNodesPerDay,
		},
		{
			start:   500,
			stop:    1000,
			wantLen: 1000,
			wantCap: 500 + approxNodesPerDay,
		},
		{
			start:   1000,
			stop:    2000,
			wantLen: 2000,
			wantCap: 2000 + approxNodesPerDay,
		},
	}

	for num, c := range cases {
		for i := c.start; i < c.stop; i++ {
			node := &BlockNode{
				Height: i,
				Hash:   bc.Hash{V0: i},
				Parent: lastNode,
			}
			blockIndex.AddNode(node)
			lastNode = node
		}
		blockIndex.SetMainChain(lastNode)
		if c.wantLen != len(blockIndex.mainChain) || c.wantCap != cap(blockIndex.mainChain) {
			t.Fatalf("SetMainChain extended capacity error, index: %d, got len: %d, got cap: %d, want len: %d, want cap: %d", num, len(blockIndex.mainChain), cap(blockIndex.mainChain), c.wantLen, c.wantCap)
		}
	}

	for i := 0; i < len(blockIndex.mainChain); i++ {
		if blockIndex.mainChain[i] != blockIndex.index[blockIndex.mainChain[i].Hash] {
			t.Fatal("SetMainChain extended capacity error, index:", i, "want:", spew.Sdump(blockIndex.mainChain[i]), "got:", spew.Sdump(blockIndex.index[blockIndex.mainChain[i].Hash]))
		}
	}
}
