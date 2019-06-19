package state

import (
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

// BlockNode represents a block within the block chain and is primarily used to
// aid in selecting the best chain to be the main chain.
type BlockNode struct {
	Parent *bc.Hash // parent is the parent block for this node.
	Hash   bc.Hash  // hash of the block.

	Version                uint64
	Height                 uint64
	Timestamp              uint64
	BlockWitness           types.BlockWitness
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
		BlockWitness:           bh.BlockWitness,
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
		BlockWitness: node.BlockWitness,
	}
}
