package database

import (
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/common"

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
	blockStoreKey          = []byte("blockStore")
	blockHeaderPrefix      = []byte("BH:")
	blockTransactonsPrefix = []byte("BTXS:")
	txStatusPrefix         = []byte("BTS:")
	voteResultPrefix       = []byte("VR:")
)

func loadBlockStoreStateJSON(db dbm.DB) *protocol.BlockStoreState {
	bytes := db.Get(blockStoreKey)
	if bytes == nil {
		return nil
	}
	bsj := &protocol.BlockStoreState{}
	if err := json.Unmarshal(bytes, bsj); err != nil {
		common.PanicCrisis(common.Fmt("Could not unmarshal bytes: %X", bytes))
	}
	return bsj
}

// A Store encapsulates storage for blockchain validation.
// It satisfies the interface protocol.Store, and provides additional
// methods for querying current data.
type Store struct {
	db    dbm.DB
	cache blockCache
}

func calcBlockHeaderKey(height uint64, hash *bc.Hash) []byte {
	buf := [8]byte{}
	binary.BigEndian.PutUint64(buf[:], height)
	key := append(blockHeaderPrefix, buf[:]...)
	return append(key, hash.Bytes()...)
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

// GetBlockHeader return the block header by given hash and height
func GetBlockHeader(db dbm.DB, hash *bc.Hash, height uint64) (*types.BlockHeader, error) {
	block := &types.Block{}
	binaryBlockHeader := db.Get(calcBlockHeaderKey(height, hash))
	if binaryBlockHeader == nil {
		return nil, nil
	}
	if err := block.UnmarshalText(binaryBlockHeader); err != nil {
		return nil, err
	}

	return &block.BlockHeader, nil
}

// GetBlockTransactions return the block transactions by given hash
func GetBlockTransactions(db dbm.DB, hash *bc.Hash) ([]*types.Tx, error) {
	block := &types.Block{}
	binaryBlockTxs := db.Get(calcBlockTransactionsKey(hash))
	if binaryBlockTxs == nil {
		return nil, errors.New("The transactions in the block is empty")
	}

	if err := block.UnmarshalText(binaryBlockTxs); err != nil {
		return nil, err
	}
	return block.Transactions, nil
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
	fillBlockHeaderFn := func(hash *bc.Hash, height uint64) (*types.BlockHeader, error) {
		return GetBlockHeader(db, hash, height)
	}
	fillBlockTxsFn := func(hash *bc.Hash) ([]*types.Tx, error) {
		return GetBlockTransactions(db, hash)
	}
	fillVoteResultFn := func(seq uint64) (*state.VoteResult, error) {
		return GetVoteResult(db, seq)
	}
	bc := newBlockCache(fillBlockHeaderFn, fillBlockTxsFn, fillVoteResultFn)
	return &Store{
		db:    db,
		cache: bc,
	}
}

// GetUtxo will search the utxo in db
func (s *Store) GetUtxo(hash *bc.Hash) (*storage.UtxoEntry, error) {
	return getUtxo(s.db, hash)
}

// BlockExist check if the block is stored in disk
func (s *Store) BlockExist(hash *bc.Hash, height uint64) bool {
	blockHeader, err := s.cache.lookupBlockHeader(hash, height)
	return err == nil && blockHeader != nil
}

// GetBlock return the block by given hash
func (s *Store) GetBlock(hash *bc.Hash, height uint64) (*types.Block, error) {
	blockHeader, err := s.GetBlockHeader(hash, height)
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
func (s *Store) GetBlockHeader(hash *bc.Hash, height uint64) (*types.BlockHeader, error) {
	blockHeader, err := s.cache.lookupBlockHeader(hash, height)
	if err != nil {
		return nil, err
	}
	return blockHeader, nil
}

// GetBlockTransactions return the Block transactions by given hash
func (s *Store) GetBlockTransactions(hash *bc.Hash) ([]*types.Tx, error) {
	txs, err := s.cache.lookupBlockTxs(hash)
	if err != nil {
		return nil, err
	}
	return txs, nil
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

// GetStoreStatus return the BlockStoreStateJSON
func (s *Store) GetStoreStatus() *protocol.BlockStoreState {
	return loadBlockStoreStateJSON(s.db)
}

// GetVoteResult retrive the voting result in specified vote sequence
func (s *Store) GetVoteResult(seq uint64) (*state.VoteResult, error) {
	return s.cache.lookupVoteResult(seq)
}

func (s *Store) LoadBlockIndex(stateBestHeight uint64) (*state.BlockIndex, error) {
	startTime := time.Now()
	blockIndex := state.NewBlockIndex()
	bhIter := s.db.IteratorPrefix(blockHeaderPrefix)
	defer bhIter.Release()

	var lastNode *state.BlockNode
	for bhIter.Next() {
		bh := &types.BlockHeader{}
		if err := bh.UnmarshalText(bhIter.Value()); err != nil {
			return nil, err
		}

		// If a block with a height greater than the best height of state is added to the index,
		// It may cause a bug that the new block cant not be process properly.
		if bh.Height > stateBestHeight {
			break
		}

		var parent *state.BlockNode
		if lastNode == nil || lastNode.Hash == bh.PreviousBlockHash {
			parent = lastNode
		} else {
			parent = blockIndex.GetNode(&bh.PreviousBlockHash)
		}

		node, err := state.NewBlockNode(bh, parent)
		if err != nil {
			return nil, err
		}

		blockIndex.AddNode(node)
		lastNode = node
	}

	log.WithFields(log.Fields{
		"module":   logModule,
		"height":   stateBestHeight,
		"duration": time.Since(startTime),
	}).Debug("initialize load history block index from database")
	return blockIndex, nil
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
	batch := s.db.NewBatch()
	batch.Set(calcBlockHeaderKey(block.Height, &blockHash), binaryBlockHeader)
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
	startTime := time.Now()

	binaryBlockHeader, err := blockHeader.MarshalText()
	if err != nil {
		return errors.Wrap(err, "Marshal block header")
	}

	blockHash := blockHeader.Hash()
	s.db.Set(calcBlockHeaderKey(blockHeader.Height, &blockHash), binaryBlockHeader)

	// updata blockheader cache
	if _, ok := s.cache.getBlockHeader(&blockHash); ok {
		s.cache.addBlockHeader(blockHeader)
	}

	log.WithFields(log.Fields{
		"module":   logModule,
		"height":   blockHeader.Height,
		"hash":     blockHash.String(),
		"duration": time.Since(startTime),
	}).Info("blockHeader saved on disk")
	return nil
}

// SaveChainStatus save the core's newest status && delete old status
func (s *Store) SaveChainStatus(node, irreversibleNode *state.BlockNode, view *state.UtxoViewpoint, voteResults []*state.VoteResult) error {
	batch := s.db.NewBatch()
	if err := saveUtxoView(batch, view); err != nil {
		return err
	}

	if err := saveVoteResult(batch, voteResults); err != nil {
		return err
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

// saveVoteResult update the voting results generated by each irreversible block
func saveVoteResult(batch dbm.Batch, voteResults []*state.VoteResult) error {
	for _, vote := range voteResults {
		bytes, err := json.Marshal(vote)
		if err != nil {
			return err
		}

		batch.Set(calcVoteResultKey(vote.Seq), bytes)
	}
	return nil
}
