package database

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"

	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/database/storage"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
)

const (
	// log module
	logModule = "leveldb"
	// the byte of colon(:)
	colon = byte(0x3a)
)

const (
	blockStore byte = iota
	blockHashes
	blockHeader
	blockTransactons
	mainChainIndex
	txStatus
	consensusResult
)

func loadBlockStoreStateJSON(db dbm.DB) *protocol.BlockStoreState {
	bytes := db.Get([]byte{blockStore})
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
	db    dbm.DB
	cache *cache
}

func calcMainChainIndexPrefix(height uint64) []byte {
	buf := [8]byte{}
	binary.BigEndian.PutUint64(buf[:], height)
	return append([]byte{mainChainIndex, colon}, buf[:]...)
}

func calcBlockHashesPrefix(height uint64) []byte {
	buf := [8]byte{}
	binary.BigEndian.PutUint64(buf[:], height)
	return append([]byte{blockHashes, colon}, buf[:]...)
}

func calcBlockHeaderKey(hash *bc.Hash) []byte {
	return append([]byte{blockHeader, colon}, hash.Bytes()...)
}

func calcBlockTransactionsKey(hash *bc.Hash) []byte {
	return append([]byte{blockTransactons, colon}, hash.Bytes()...)
}

func calcTxStatusKey(hash *bc.Hash) []byte {
	return append([]byte{txStatus, colon}, hash.Bytes()...)
}

func calcConsensusResultKey(seq uint64) []byte {
	buf := [8]byte{}
	binary.BigEndian.PutUint64(buf[:], seq)
	return append([]byte{consensusResult, colon}, buf[:]...)
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

// GetBlockHashesByHeight return block hashes by given height
func GetBlockHashesByHeight(db dbm.DB, height uint64) ([]*bc.Hash, error) {
	binaryHashes := db.Get(calcBlockHashesPrefix(height))
	if binaryHashes == nil {
		return []*bc.Hash{}, nil
	}

	hashes := []*bc.Hash{}
	if err := json.Unmarshal(binaryHashes, &hashes); err != nil {
		return nil, err
	}
	return hashes, nil
}

// GetMainChainHash return BlockHash by given height
func GetMainChainHash(db dbm.DB, height uint64) (*bc.Hash, error) {
	binaryHash := db.Get(calcMainChainIndexPrefix(height))
	if binaryHash == nil {
		return nil, fmt.Errorf("There are no BlockHash with given height %d", height)
	}

	hash := &bc.Hash{}
	if err := hash.UnmarshalText(binaryHash); err != nil {
		return nil, err
	}
	return hash, nil
}

// GetConsensusResult return the vote result by given sequence
func GetConsensusResult(db dbm.DB, seq uint64) (*state.ConsensusResult, error) {
	data := db.Get(calcConsensusResultKey(seq))
	if data == nil {
		return nil, protocol.ErrNotFoundConsensusResult
	}

	consensusResult := new(state.ConsensusResult)
	if err := json.Unmarshal(data, consensusResult); err != nil {
		return nil, errors.Wrap(err, "unmarshaling vote result")
	}
	return consensusResult, nil
}

// DeleteConsensusResult delete a consensusResult from cache and database
func (s *Store) DeleteConsensusResult(seq uint64) error {
	consensusResult, err := GetConsensusResult(s.db, seq)
	if err != nil {
		return err
	}

	s.db.Delete(calcConsensusResultKey(seq))
	s.cache.removeConsensusResult(consensusResult)
	return nil
}

// DeleteBlock delete a new block in the protocol.
func (s *Store) DeleteBlock(block *types.Block) error {
	blockHash := block.Hash()
	blockHashes, err := s.GetBlockHashesByHeight(block.Height)
	if err != nil {
		return err
	}

	for i := 0; i < len(blockHashes); i++ {
		if blockHashes[i].String() == blockHash.String() {
			blockHashes = append(blockHashes[0:i], blockHashes[i+1:len(blockHashes)]...)
			break
		}
	}

	batch := s.db.NewBatch()
	if len(blockHashes) == 0 {
		batch.Delete(calcBlockHashesPrefix(block.Height))
	} else {
		binaryBlockHashes, err := json.Marshal(blockHashes)
		if err != nil {
			return errors.Wrap(err, "Marshal block hashes")
		}

		batch.Set(calcBlockHashesPrefix(block.Height), binaryBlockHashes)
	}

	batch.Delete(calcBlockHeaderKey(&blockHash))
	batch.Delete(calcBlockTransactionsKey(&blockHash))
	batch.Delete(calcTxStatusKey(&blockHash))
	batch.Write()

	s.cache.removeBlockHashes(block.Height)
	s.cache.removeBlockHeader(&block.BlockHeader)

	return nil
}

// NewStore creates and returns a new Store object.
func NewStore(db dbm.DB) *Store {
	fillBlockHeaderFn := func(hash *bc.Hash) (*types.BlockHeader, error) {
		return GetBlockHeader(db, hash)
	}
	fillBlockTxsFn := func(hash *bc.Hash) ([]*types.Tx, error) {
		return GetBlockTransactions(db, hash)
	}

	fillBlockHashesFn := func(height uint64) ([]*bc.Hash, error) {
		return GetBlockHashesByHeight(db, height)
	}

	fillMainChainHashFn := func(height uint64) (*bc.Hash, error) {
		return GetMainChainHash(db, height)
	}

	fillConsensusResultFn := func(seq uint64) (*state.ConsensusResult, error) {
		return GetConsensusResult(db, seq)
	}

	cache := newCache(fillBlockHeaderFn, fillBlockTxsFn, fillBlockHashesFn, fillMainChainHashFn, fillConsensusResultFn)
	return &Store{
		db:    db,
		cache: cache,
	}
}

// BlockExist check if the block is stored in disk
func (s *Store) BlockExist(hash *bc.Hash) bool {
	_, err := s.cache.lookupBlockHeader(hash)
	return err == nil
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

// GetBlockHashesByHeight return the block hash by the specified height
func (s *Store) GetBlockHashesByHeight(height uint64) ([]*bc.Hash, error) {
	return s.cache.lookupBlockHashesByHeight(height)
}

// GetMainChainHash return the block hash by the specified height
func (s *Store) GetMainChainHash(height uint64) (*bc.Hash, error) {
	return s.cache.lookupMainChainHash(height)
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

// GetConsensusResult retrive the voting result in specified vote sequence
func (s *Store) GetConsensusResult(seq uint64) (*state.ConsensusResult, error) {
	return s.cache.lookupConsensusResult(seq)
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
		return errors.Wrap(err, "Marshal block transaction status")
	}

	blockHashes := []*bc.Hash{}
	hashes, err := s.GetBlockHashesByHeight(block.Height)
	if err != nil {
		return err
	}
	blockHashes = append(blockHashes, hashes...)
	blockHash := block.Hash()
	blockHashes = append(blockHashes, &blockHash)
	binaryBlockHashes, err := json.Marshal(blockHashes)
	if err != nil {
		return errors.Wrap(err, "Marshal block hashes")
	}

	batch := s.db.NewBatch()
	batch.Set(calcBlockHashesPrefix(block.Height), binaryBlockHashes)
	batch.Set(calcBlockHeaderKey(&blockHash), binaryBlockHeader)
	batch.Set(calcBlockTransactionsKey(&blockHash), binaryBlockTxs)
	batch.Set(calcTxStatusKey(&blockHash), binaryTxStatus)
	batch.Write()

	s.cache.removeBlockHashes(block.Height)
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
func (s *Store) SaveChainStatus(blockHeader, irrBlockHeader *types.BlockHeader, mainBlockHeaders []*types.BlockHeader, view *state.UtxoViewpoint, consensusResults []*state.ConsensusResult) error {
	currentStatus := loadBlockStoreStateJSON(s.db)
	batch := s.db.NewBatch()
	if err := saveUtxoView(batch, view); err != nil {
		return err
	}

	var clearCacheFuncs []func()
	for _, consensusResult := range consensusResults {
		result := consensusResult
		bytes, err := json.Marshal(result)
		if err != nil {
			return err
		}

		batch.Set(calcConsensusResultKey(result.Seq), bytes)
		clearCacheFuncs = append(clearCacheFuncs, func() {
			s.cache.removeConsensusResult(result)
		})
	}

	blockHash := blockHeader.Hash()
	irrBlockHash := irrBlockHeader.Hash()
	bytes, err := json.Marshal(protocol.BlockStoreState{
		Height:             blockHeader.Height,
		Hash:               &blockHash,
		IrreversibleHeight: irrBlockHeader.Height,
		IrreversibleHash:   &irrBlockHash,
	})
	if err != nil {
		return err
	}
	batch.Set([]byte{blockStore}, bytes)

	// save main chain blockHeaders
	for _, blockHeader := range mainBlockHeaders {
		bh := blockHeader
		blockHash := bh.Hash()
		binaryBlockHash, err := blockHash.MarshalText()
		if err != nil {
			return errors.Wrap(err, "Marshal block hash")
		}

		batch.Set(calcMainChainIndexPrefix(bh.Height), binaryBlockHash)
		clearCacheFuncs = append(clearCacheFuncs, func() {
			s.cache.removeMainChainHash(bh.Height)
		})
	}

	if currentStatus != nil {
		for i := blockHeader.Height + 1; i <= currentStatus.Height; i++ {
			index := i
			batch.Delete(calcMainChainIndexPrefix(index))
			clearCacheFuncs = append(clearCacheFuncs, func() {
				s.cache.removeMainChainHash(index)
			})
		}
	}
	batch.Write()

	for _, clearCacheFunc := range clearCacheFuncs {
		clearCacheFunc()
	}
	return nil
}
