package database

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"

	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/database/storage"
	"github.com/vapor/errors"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
)

const logModule = "leveldb"

var (
	blockStoreKey           = []byte("blockStore")
	blockHashByHeightPrefix = []byte("BHH:")
	blockHeaderPrefix       = []byte("BH:")
	blockTransactonsPrefix  = []byte("BTXS:")
	txStatusPrefix          = []byte("BTS:")
	voteResultPrefix        = []byte("VR:")
)

func loadBlockStoreStateJSON(db dbm.DB) *protocol.BlockStoreState {
	bytes := db.Get(blockStoreKey)
	if bytes == nil {
		return nil
	}

	bsj := &protocol.BlockStoreState{}
	if err := json.Unmarshal(bytes, bsj); err != nil {
		log.WithField("err", err).Panic("fail on unmarshal BlockStoreStateJSON")
	}
	return bsj
}

// A Store encapsulates storage for blockchain validation.
// It satisfies the interface protocol.Store, and provides additional
// methods for querying current data.
type Store struct {
	db         dbm.DB
	cache      cache
	blockIndex *state.BlockIndex
}

func calcBlockHashByHeightKey(height uint64) []byte {
	buf := [8]byte{}
	binary.BigEndian.PutUint64(buf[:], height)
	return append(blockHashByHeightPrefix, buf[:]...)
}

func calcBlockHeaderKey(hash *bc.Hash) []byte {
	return append(blockHeaderPrefix, hash.Bytes()...)
}

func calcBlockTransactionsKey(hash *bc.Hash) []byte {
	return append(blockTransactonsPrefix, hash.Bytes()...)
}

func calcTxStatusKey(hash *bc.Hash) []byte {
	return append(txStatusPrefix, hash.Bytes()...)
}

func calcVoteResultKey(seq uint64) []byte {
	buf := [8]byte{}
	binary.BigEndian.PutUint64(buf[:], seq)
	return append(voteResultPrefix, buf[:]...)
}

// GetBlockHeader return the block header by given hash
func GetBlockHeader(db dbm.DB, hash *bc.Hash) (*types.BlockHeader, error) {
	binaryBlockHeader := db.Get(calcBlockHeaderKey(hash))
	if binaryBlockHeader == nil {
		return nil, fmt.Errorf("There are no blockHeader with given hash %s", hash.String())
	}

	blockHeader := &types.BlockHeader{}
	if err := blockHeader.UnmarshalText(binaryBlockHeader); err != nil {
		return nil, err
	}
	return blockHeader, nil
}

// GetBlockTransactions return the block transactions by given hash
func GetBlockTransactions(db dbm.DB, hash *bc.Hash) ([]*types.Tx, error) {
	binaryBlockTxs := db.Get(calcBlockTransactionsKey(hash))
	if binaryBlockTxs == nil {
		return nil, fmt.Errorf("There are no block transactions with given hash %s", hash.String())
	}

	block := &types.Block{}
	if err := block.UnmarshalText(binaryBlockTxs); err != nil {
		return nil, err
	}
	return block.Transactions, nil
}

func GetBlockNode(db dbm.DB, hash *bc.Hash) (*state.BlockNode, error) {
	blockHeader, err := GetBlockHeader(db, hash)
	if err != nil {
		return nil, err
	}
	return state.NewBlockNode(blockHeader)
}

// GetVoteResult return the vote result by given sequence
func GetVoteResult(db dbm.DB, seq uint64) (*state.VoteResult, error) {
	data := db.Get(calcVoteResultKey(seq))
	if data == nil {
		return nil, protocol.ErrNotFoundVoteResult
	}

	voteResult := new(state.VoteResult)
	if err := json.Unmarshal(data, voteResult); err != nil {
		return nil, errors.Wrap(err, "unmarshaling vote result")
	}
	return voteResult, nil
}

// NewStore creates and returns a new Store object.
func NewStore(db dbm.DB) *Store {
	fillBlockHeaderFn := func(hash *bc.Hash) (*types.BlockHeader, error) {
		return GetBlockHeader(db, hash)
	}
	fillBlockTxsFn := func(hash *bc.Hash) ([]*types.Tx, error) {
		return GetBlockTransactions(db, hash)
	}

	fillVoteResultFn := func(seq uint64) (*state.VoteResult, error) {
		return GetVoteResult(db, seq)
	}

	fillBlockNodeFn := func(hash *bc.Hash) (*state.BlockNode, error) {
		return GetBlockNode(db, hash)
	}

	cache := newCache(fillBlockHeaderFn, fillBlockTxsFn, fillVoteResultFn)
	blockIndex := state.NewBlockIndex(fillBlockNodeFn)
	return &Store{
		db:         db,
		cache:      cache,
		blockIndex: blockIndex,
	}
}

// BlockExist check if the block is stored in disk
func (s *Store) BlockExist(hash *bc.Hash) bool {
	blockHeader, err := s.cache.lookupBlockHeader(hash)
	return err == nil && blockHeader != nil
}

// GetBlock return the block by given hash
func (s *Store) GetBlock(hash *bc.Hash) (*types.Block, error) {
	blockHeader, err := s.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	txs, err := s.GetBlockTransactions(hash)
	if err != nil {
		return nil, err
	}

	return &types.Block{
		BlockHeader:  *blockHeader,
		Transactions: txs,
	}, nil
}

// GetBlockHeader return the BlockHeader by given hash
func (s *Store) GetBlockHeader(hash *bc.Hash) (*types.BlockHeader, error) {
	return s.cache.lookupBlockHeader(hash)
}

// GetBlockTransactions return the Block transactions by given hash
func (s *Store) GetBlockTransactions(hash *bc.Hash) ([]*types.Tx, error) {
	return s.cache.lookupBlockTxs(hash)
}

// GetStoreStatus return the BlockStoreStateJSON
func (s *Store) GetStoreStatus() *protocol.BlockStoreState {
	return loadBlockStoreStateJSON(s.db)
}

// GetTransactionsUtxo will return all the utxo that related to the input txs
func (s *Store) GetTransactionsUtxo(view *state.UtxoViewpoint, txs []*bc.Tx) error {
	return getTransactionsUtxo(s.db, view, txs)
}

// GetTransactionStatus will return the utxo that related to the block hash
func (s *Store) GetTransactionStatus(hash *bc.Hash) (*bc.TransactionStatus, error) {
	data := s.db.Get(calcTxStatusKey(hash))
	if data == nil {
		return nil, errors.New("can't find the transaction status by given hash")
	}

	ts := &bc.TransactionStatus{}
	if err := proto.Unmarshal(data, ts); err != nil {
		return nil, errors.Wrap(err, "unmarshaling transaction status")
	}
	return ts, nil
}

// GetUtxo will search the utxo in db
func (s *Store) GetUtxo(hash *bc.Hash) (*storage.UtxoEntry, error) {
	return getUtxo(s.db, hash)
}

// GetVoteResult retrive the voting result in specified vote sequence
func (s *Store) GetVoteResult(seq uint64) (*state.VoteResult, error) {
	return s.cache.lookupVoteResult(seq)
}

func (s *Store) LoadBlockIndex(stateBestHeight uint64) (*state.BlockIndex, error) {
	startTime := time.Now()
	bhIter := s.db.IteratorPrefix(blockHashByHeightPrefix)
	defer bhIter.Release()

	for bhIter.Next() {
		key := bhIter.Key()
		lenPrefix := len(blockHashByHeightPrefix)
		blockNodeHeight := binary.BigEndian.Uint64(key[lenPrefix:])

		// If a block with a height greater than the best height of state is added to the index,
		// It may cause a bug that the new block cant not be process properly.
		if blockNodeHeight > stateBestHeight {
			break
		}

		blockNodeHash := &bc.Hash{}
		if err := blockNodeHash.UnmarshalText(bhIter.Value()); err != nil {
			return nil, err
		}

		blockHeader, err := GetBlockHeader(s.db, blockNodeHash)
		if err != nil {
			return nil, err
		}

		node, err := state.NewBlockNode(blockHeader)
		if err != nil {
			return nil, err
		}
		s.blockIndex.AddNode(node)
	}

	log.WithFields(log.Fields{
		"module":   logModule,
		"height":   stateBestHeight,
		"duration": time.Since(startTime),
	}).Debug("initialize load history block index from database")
	return s.blockIndex, nil
}

// SaveBlock persists a new block in the protocol.
func (s *Store) SaveBlock(block *types.Block, ts *bc.TransactionStatus) error {
	startTime := time.Now()
	binaryBlockHeader, err := block.MarshalTextForBlockHeader()
	if err != nil {
		return errors.Wrap(err, "Marshal block header")
	}

	binaryBlockTxs, err := block.MarshalTextForTransactions()
	if err != nil {
		return errors.Wrap(err, "Marshal block transactions")
	}

	binaryTxStatus, err := proto.Marshal(ts)
	if err != nil {
		return errors.Wrap(err, "marshal block transaction status")
	}

	blockHash := block.Hash()
	binaryBlockHash, err := blockHash.MarshalText()
	if err != nil {
		return errors.Wrap(err, "marshal block hash")
	}

	batch := s.db.NewBatch()
	batch.Set(calcBlockHashByHeightKey(block.Height), binaryBlockHash)
	batch.Set(calcBlockHeaderKey(&blockHash), binaryBlockHeader)
	batch.Set(calcBlockTransactionsKey(&blockHash), binaryBlockTxs)
	batch.Set(calcTxStatusKey(&blockHash), binaryTxStatus)
	batch.Write()

	log.WithFields(log.Fields{
		"module":   logModule,
		"height":   block.Height,
		"hash":     blockHash.String(),
		"duration": time.Since(startTime),
	}).Info("block saved on disk")
	return nil
}

// SaveBlockHeader persists a new block header in the protocol.
func (s *Store) SaveBlockHeader(blockHeader *types.BlockHeader) error {
	binaryBlockHeader, err := blockHeader.MarshalText()
	if err != nil {
		return errors.Wrap(err, "Marshal block header")
	}

	blockHash := blockHeader.Hash()
	s.db.Set(calcBlockHeaderKey(&blockHash), binaryBlockHeader)
	s.cache.removeBlockHeader(blockHeader)
	return nil
}

// SaveChainStatus save the core's newest status && delete old status
func (s *Store) SaveChainStatus(node, irreversibleNode *state.BlockNode, view *state.UtxoViewpoint, voteResults []*state.VoteResult) error {
	batch := s.db.NewBatch()
	if err := saveUtxoView(batch, view); err != nil {
		return err
	}

	for _, vote := range voteResults {
		bytes, err := json.Marshal(vote)
		if err != nil {
			return err
		}

		batch.Set(calcVoteResultKey(vote.Seq), bytes)
		s.cache.removeVoteResult(vote)
	}

	bytes, err := json.Marshal(protocol.BlockStoreState{
		Height:             node.Height,
		Hash:               &node.Hash,
		IrreversibleHeight: irreversibleNode.Height,
		IrreversibleHash:   &irreversibleNode.Hash,
	})
	if err != nil {
		return err
	}

	batch.Set(blockStoreKey, bytes)
	batch.Write()
	return nil
}
