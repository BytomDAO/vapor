package bench

import (
	"github.com/vapor/database/storage"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
)

type Store struct {
	BlockStore       *protocol.BlockStoreState
	Blocks           map[bc.Hash]*types.Block
	BlockHeaders     map[bc.Hash]*types.BlockHeader
	BlockHashes      map[uint64][]*bc.Hash
	MainChainHashes  map[uint64]*bc.Hash
	TxStatuses       map[bc.Hash]*bc.TransactionStatus
	ConsensusResults map[uint64]*state.ConsensusResult
	Entries          map[bc.Hash]*storage.UtxoEntry
}

func (s *Store) BlockExist(hash *bc.Hash) bool {
	_, ok := s.Blocks[*hash]
	return ok
}

func (s *Store) GetBlock(hash *bc.Hash) (*types.Block, error) {
	return s.Blocks[*hash], nil
}

func (s *Store) GetBlockHeader(hash *bc.Hash) (*types.BlockHeader, error) {
	return s.BlockHeaders[*hash], nil
}

func (s *Store) GetStoreStatus() *protocol.BlockStoreState {
	return s.BlockStore
}

func (s *Store) GetTransactionStatus(hash *bc.Hash) (*bc.TransactionStatus, error) {
	return s.TxStatuses[*hash], nil
}

func (s *Store) GetTransactionsUtxo(view *state.UtxoViewpoint, txs []*bc.Tx) error {
	view.Entries = s.Entries
	return nil
}

func (s *Store) GetUtxo(hash *bc.Hash) (*storage.UtxoEntry, error) {
	return s.Entries[*hash], nil
}

func (s *Store) GetConsensusResult(seq uint64) (*state.ConsensusResult, error) {
	return s.ConsensusResults[seq], nil
}

func (s *Store) GetMainChainHash(height uint64) (*bc.Hash, error) {
	return s.MainChainHashes[height], nil
}

func (s *Store) GetBlockHashesByHeight(height uint64) ([]*bc.Hash, error) {
	return s.BlockHashes[height], nil
}

func (s *Store) SaveBlock(block *types.Block, txStatus *bc.TransactionStatus) error {
	blockHash := block.Hash()
	s.Blocks[blockHash] = block
	s.BlockHeaders[blockHash] = &block.BlockHeader
	s.TxStatuses[blockHash] = txStatus
	s.BlockHashes[block.Height] = append(s.BlockHashes[block.Height], &blockHash)
	s.MainChainHashes[block.Height] = &blockHash
	return nil
}

func (s *Store) SaveBlockHeader(blockHeader *types.BlockHeader) error {
	s.BlockHeaders[blockHeader.Hash()] = blockHeader
	return nil
}

func (s *Store) SaveChainStatus(blockHeader, irrBlockHeader *types.BlockHeader, mainBlockHeaders []*types.BlockHeader, view *state.UtxoViewpoint, consensusResults []*state.ConsensusResult) error {
	// save utxo view
	for key, entry := range view.Entries {
		if (entry.Type == storage.CrosschainUTXOType) && (!entry.Spent) {
			if _, ok := s.Entries[key]; ok {
				delete(s.Entries, key)
			}
			continue
		}

		if (entry.Type == storage.NormalUTXOType || entry.Type == storage.VoteUTXOType) && (entry.Spent) {
			if _, ok := s.Entries[key]; ok {
				delete(s.Entries, key)
			}
			continue
		}
		s.Entries[key] = entry
	}

	// save vote result
	for _, vote := range consensusResults {
		s.ConsensusResults[vote.Seq] = vote
	}

	// save block store
	blockHash := blockHeader.Hash()
	irrBlockHash := irrBlockHeader.Hash()
	s.BlockStore = &protocol.BlockStoreState{
		Height:             blockHeader.Height,
		Hash:               &blockHash,
		IrreversibleHeight: irrBlockHeader.Height,
		IrreversibleHash:   &irrBlockHash,
	}

	// save reorganize block hash
	for _, bh := range mainBlockHeaders {
		blockHash := bh.Hash()
		s.MainChainHashes[bh.Height] = &blockHash
	}
	return nil
}

func NewStore() *Store {
	return &Store{
		Blocks:           make(map[bc.Hash]*types.Block),
		BlockHeaders:     make(map[bc.Hash]*types.BlockHeader),
		BlockHashes:      make(map[uint64][]*bc.Hash),
		MainChainHashes:  make(map[uint64]*bc.Hash),
		TxStatuses:       make(map[bc.Hash]*bc.TransactionStatus),
		ConsensusResults: make(map[uint64]*state.ConsensusResult),
		Entries:          make(map[bc.Hash]*storage.UtxoEntry),
	}
}
