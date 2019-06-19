package state

import (
	"github.com/vapor/common"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	// approxNodesPerDay is the approximate number of new blocks in a day on average.
	approxNodesPerDay = 2 * 24 * 60 * 60
	// maxCachedMainBlockNodes is the max number of cached blockNodes
	maxCachedBlockNodes = 10000
	// maxCachedHeightIndexes is the max number of cached blockNodes
	maxCachedHeightIndexes = 10000
	// maxCachedMainChainHashes is the max number of cached blockNodes
	maxCachedMainChainHashes = 10000
)

type fillBlockNodeFn func(hash *bc.Hash) (*BlockNode, error)
type fillHeightIndexFn func(height uint64) ([]*bc.Hash, error)
type fillMainChainHashFn func(height uint64) (*bc.Hash, error)

// BlockNode represents a block within the block chain and is primarily used to
// aid in selecting the best chain to be the main chain.
type BlockNode struct {
	Parent *bc.Hash // parent is the parent block for this node.
	Hash   bc.Hash  // hash of the block.

	Version                uint64
	Height                 uint64
	Timestamp              uint64
	BlockWitness           [][]byte
	TransactionsMerkleRoot bc.Hash
	TransactionStatusHash  bc.Hash
}

// NewBlockNode create a BlockNode
func NewBlockNode(bh *types.BlockHeader) *BlockNode {
	return &BlockNode{
		Parent:                 &bh.PreviousBlockHash,
		Hash:                   bh.Hash(),
		Version:                bh.Version,
		Height:                 bh.Height,
		Timestamp:              bh.Timestamp,
		BlockWitness:           bh.Witness,
		TransactionsMerkleRoot: bh.TransactionsMerkleRoot,
		TransactionStatusHash:  bh.TransactionStatusHash,
	}
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
	lruBlockNodes      *common.Cache
	lruHeightIndexes   *common.Cache
	lruMainChainHashes *common.Cache

	fillBlockNodeFn     func(*bc.Hash) (*BlockNode, error)
	fillHeightIndexFn   func(uint64) ([]*bc.Hash, error)
	fillMainChainHashFn func(uint64) (*bc.Hash, error)
}

// NewBlockIndex create a BlockIndex
func NewBlockIndex(fillBlockNode fillBlockNodeFn, fillHeightIndex fillHeightIndexFn, fillMainChainHash fillMainChainHashFn) *BlockIndex {
	return &BlockIndex{
		lruBlockNodes:      common.NewCache(maxCachedBlockNodes),
		lruHeightIndexes:   common.NewCache(maxCachedHeightIndexes),
		lruMainChainHashes: common.NewCache(maxCachedMainChainHashes),

		fillBlockNodeFn:     fillBlockNode,
		fillHeightIndexFn:   fillHeightIndex,
		fillMainChainHashFn: fillMainChainHash,
	}
}

// GetNode search BlockNode from the index map
func (bi *BlockIndex) GetBlockNode(hash *bc.Hash) (*BlockNode, error) {
	if hexBlockNode, ok := bi.lruBlockNodes.Get(*hash); ok {
		return hexBlockNode.(*BlockNode), nil
	}

	blockNode, err := bi.fillBlockNodeFn(hash)
	if err != nil {
		return nil, nil
	}
	bi.lruBlockNodes.Add(hash, blockNode)
	return blockNode, nil
}

// BestNode return the best BlockNode
// func (bi *BlockIndex) BestNode() *BlockNode {
// 	return bi.bestNode
// }

// BlockExist check does the block existed in blockIndex
func (bi *BlockIndex) BlockExist(hash *bc.Hash) bool {
	if _, ok := bi.lruBlockNodes.Get(*hash); ok {
		return ok
	}

	if _, err := bi.fillBlockNodeFn(hash); err != nil {
		return true
	}
	return false
}

// TODO: THIS FUNCTION MIGHT BE DELETED
// func (bi *BlockIndex) InMainchain(hash *bc.Hash) bool {
// 	if resBlockNode, ok := bi.lruBlockNodes.Get(*hash); ok {
// 		blockNode := resBlockNode.(*BlockNode)
// 		return *bi.mainChain[blockNode.Height] == *hash
// 	}

// 	if blockNode, err := bi.fillBlockNodeFn(hash); err != nil {
// 		return *bi.mainChain[blockNode.Height] == *hash
// 	}
// 	return false
// }

// NodeByHeight return the BlockNode at the specified height
func (bi *BlockIndex) GetBlockHashByHeight(height uint64) (*bc.Hash, error) {
	if hash, ok := bi.lruMainChainHashes.Get(height); ok {
		return hash.(*bc.Hash), nil
	}

	hash, err := bi.fillMainChainHashFn(height)
	if err != nil {
		return nil, nil
	}
	bi.lruMainChainHashes.Add(height, hash)
	return hash, nil
}

// NodesByHeight return all block nodes at the specified height.
func (bi *BlockIndex) GetBlockHashesByHeight(height uint64) ([]*bc.Hash, error) {
	if hashes, ok := bi.lruHeightIndexes.Get(height); ok {
		return hashes.([]*bc.Hash), nil
	}

	hashes, err := bi.fillHeightIndexFn(height)
	if err != nil {
		return nil, nil
	}
	bi.lruHeightIndexes.Add(height, hashes)
	return hashes, nil
}

// SetMainChain set the the mainChain array
// func (bi *BlockIndex) SetMainChain(node *BlockNode) {
// 	bi.Lock()
// 	defer bi.Unlock()

// 	needed := node.Height + 1
// 	if uint64(cap(bi.mainChain)) < needed {
// 		blockNodeHashes := make([]*bc.Hash, needed, needed+approxNodesPerDay)
// 		copy(blockNodeHashes, bi.mainChain)
// 		bi.mainChain = blockNodeHashes
// 	} else {
// 		i := uint64(len(bi.mainChain))
// 		bi.mainChain = bi.mainChain[0:needed]
// 		for ; i < needed; i++ {
// 			bi.mainChain[i] = nil
// 		}
// 	}

// 	for node != nil && bi.mainChain[node.Height].String() != node.Hash.String() {
// 		bi.mainChain[node.Height] = &node.Hash
// 		node = bi.GetNode(node.Parent)
// 	}
// }
