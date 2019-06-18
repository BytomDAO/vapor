package state

import (
	"sync"

	"github.com/vapor/common"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	// approxNodesPerDay is the approximate number of new blocks in a day on average.
	approxNodesPerDay = 2 * 24 * 60 * 60
	// maxCachedMainBlockNodes is the max number of cached blockNodes
	maxCachedMainBlockNodes = 10000
)

type fillBlockNodeFn func(hash *bc.Hash) (*BlockNode, error)

// BlockNode represents a block within the block chain and is primarily used to
// aid in selecting the best chain to be the main chain.
type BlockNode struct {
	Parent *bc.Hash // parent is the parent block for this node.
	Hash   bc.Hash  // hash of the block.

	Version                uint64
	Height                 uint64
	Timestamp              uint64
	BlockWitness           *common.BitMap
	TransactionsMerkleRoot bc.Hash
	TransactionStatusHash  bc.Hash
}

// NewBlockNode create a BlockNode
func NewBlockNode(bh *types.BlockHeader) (*BlockNode, error) {
	node := &BlockNode{
		Parent:                 &bh.PreviousBlockHash,
		Hash:                   bh.Hash(),
		Version:                bh.Version,
		Height:                 bh.Height,
		Timestamp:              bh.Timestamp,
		TransactionsMerkleRoot: bh.TransactionsMerkleRoot,
		TransactionStatusHash:  bh.TransactionStatusHash,
	}

	node.BlockWitness = common.NewBitMap(uint32(len(bh.Witness)))
	for i, witness := range bh.Witness {
		if len(witness) != 0 {
			if err := node.BlockWitness.Set(uint32(i)); err != nil {
				return nil, err
			}
		}
	}
	return node, nil
}

// BlockHeader convert a BlockNode to the BlockHeader
func (node *BlockNode) BlockHeader() *types.BlockHeader {
	previousBlockHash := bc.Hash{}
	if node.Parent != nil {
		previousBlockHash = *node.Parent
	}
	return &types.BlockHeader{
		Version:           node.Version,
		Height:            node.Height,
		PreviousBlockHash: previousBlockHash,
		Timestamp:         node.Timestamp,
		BlockCommitment: types.BlockCommitment{
			TransactionsMerkleRoot: node.TransactionsMerkleRoot,
			TransactionStatusHash:  node.TransactionStatusHash,
		},
	}
}

// BlockIndex is the struct for help chain trace block chain as tree
type BlockIndex struct {
	sync.RWMutex
	lruMainBlockNodes *common.Cache
	fillBlockNodeFn   func(*bc.Hash) (*BlockNode, error)
	heightIndex       map[uint64][]*bc.Hash
	mainChain         []*bc.Hash
}

// NewBlockIndex create a BlockIndex
func NewBlockIndex(fillBlockNode fillBlockNodeFn) *BlockIndex {
	return &BlockIndex{
		lruMainBlockNodes: common.NewCache(maxCachedMainBlockNodes),
		fillBlockNodeFn:   fillBlockNode,
		heightIndex:       make(map[uint64][]*bc.Hash),
		mainChain:         make([]*bc.Hash, 0, approxNodesPerDay),
	}
}

// AddNode add BlockNode into the index map
func (bi *BlockIndex) AddNode(node *BlockNode) {
	bi.Lock()
	bi.heightIndex[node.Height] = append(bi.heightIndex[node.Height], &node.Hash)
	bi.Unlock()
	bi.lruMainBlockNodes.Add(node.Hash, node)
}

// GetNode search BlockNode from the index map
func (bi *BlockIndex) GetNode(hash *bc.Hash) *BlockNode {
	if hexBlockNode, ok := bi.lruMainBlockNodes.Get(*hash); ok {
		return hexBlockNode.(*BlockNode)
	}

	if blockNode, err := bi.fillBlockNodeFn(hash); err != nil {
		return blockNode
	}
	return nil
}

// BestNode return the best BlockNode
func (bi *BlockIndex) BestNode() *BlockNode {
	bi.RLock()
	defer bi.RUnlock()
	bestBlockHash := bi.mainChain[len(bi.mainChain)-1]
	return bi.GetNode(bestBlockHash)
}

// BlockExist check does the block existed in blockIndex
func (bi *BlockIndex) BlockExist(hash *bc.Hash) bool {
	if _, ok := bi.lruMainBlockNodes.Get(*hash); ok {
		return ok
	}

	if _, err := bi.fillBlockNodeFn(hash); err != nil {
		return true
	}
	return false
}

// TODO: THIS FUNCTION MIGHT BE DELETED
func (bi *BlockIndex) InMainchain(hash *bc.Hash) bool {
	if resBlockNode, ok := bi.lruMainBlockNodes.Get(*hash); ok {
		blockNode := resBlockNode.(*BlockNode)
		return *bi.mainChain[blockNode.Height] == *hash
	}

	if blockNode, err := bi.fillBlockNodeFn(hash); err != nil {
		return *bi.mainChain[blockNode.Height] == *hash
	}
	return false
}

// NodeByHeight return the BlockNode at the specified height
func (bi *BlockIndex) NodeByHeight(height uint64) *BlockNode {
	bi.RLock()
	defer bi.RUnlock()
	if height >= uint64(len(bi.mainChain)) {
		return nil
	}
	return bi.GetNode(bi.mainChain[height])
}

// NodesByHeight return all block nodes at the specified height.
func (bi *BlockIndex) NodesByHeight(height uint64) []*BlockNode {
	bi.RLock()
	defer bi.RUnlock()

	blockNodeHashes := bi.heightIndex[height]
	blockNodes := []*BlockNode{}
	for _, h := range blockNodeHashes {
		blockNode := bi.GetNode(h)
		blockNodes = append(blockNodes, blockNode)
	}
	return blockNodes
}

// SetMainChain set the the mainChain array
func (bi *BlockIndex) SetMainChain(node *BlockNode) {
	bi.Lock()
	defer bi.Unlock()

	needed := node.Height + 1
	if uint64(cap(bi.mainChain)) < needed {
		blockNodeHashes := make([]*bc.Hash, needed, needed+approxNodesPerDay)
		copy(blockNodeHashes, bi.mainChain)
		bi.mainChain = blockNodeHashes
	} else {
		i := uint64(len(bi.mainChain))
		bi.mainChain = bi.mainChain[0:needed]
		for ; i < needed; i++ {
			bi.mainChain[i] = nil
		}
	}

	for node != nil && *bi.mainChain[node.Height] != node.Hash {
		bi.mainChain[node.Height] = &node.Hash
		node = bi.GetNode(node.Parent)
	}
}
