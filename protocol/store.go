package protocol

import (
	"errors"

	"github.com/bytom/vapor/database/storage"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
)

// predefine errors
var (
	ErrNotFoundConsensusResult = errors.New("can't find the vote result by given sequence")
)

// Store provides storage interface for blockchain data
type Store interface {
	BlockExist(*bc.Hash) bool

	GetBlock(*bc.Hash) (*types.Block, error)
	GetBlockHeader(*bc.Hash) (*types.BlockHeader, error)
	GetStoreStatus() *BlockStoreState
	GetTransactionStatus(*bc.Hash) (*bc.TransactionStatus, error)
	GetTransactionsUtxo(*state.UtxoViewpoint, []*bc.Tx) error
	GetUtxo(*bc.Hash) (*storage.UtxoEntry, error)
	GetConsensusResult(uint64) (*state.ConsensusResult, error)
	GetMainChainHash(uint64) (*bc.Hash, error)
	GetBlockHashesByHeight(uint64) ([]*bc.Hash, error)

	DeleteConsensusResult(uint64) error
	DeleteBlock(*types.Block) error
	SaveBlock(*types.Block, *bc.TransactionStatus) error
	SaveBlockHeader(*types.BlockHeader) error
	SaveChainStatus(*types.BlockHeader, *types.BlockHeader, []*types.BlockHeader, *state.UtxoViewpoint, []*state.ConsensusResult) error
}

// BlockStoreState represents the core's db status
type BlockStoreState struct {
	Height             uint64
	Hash               *bc.Hash
	IrreversibleHeight uint64
	IrreversibleHash   *bc.Hash
}
