package database

import (
	"time"

	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/common"

	dbm "github.com/vapor/database/db"
	"github.com/vapor/database/orm"
	"github.com/vapor/database/storage"
	"github.com/vapor/errors"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
)

const logModuleSQL = "SQLdb"

func loadBlockSQLStoreStateJSON(db dbm.SQLDB) *protocol.BlockStoreState {
	bsj := orm.BlockStoreState{
		StoreKey: string(blockStoreKey),
	}

	SQLDB := db.Db()
	//if err := SQLDB.Where("store_key = ?", string(blockStoreKey)).First(&bsj).Error; err != nil {
	if err := SQLDB.Where(&bsj).First(&bsj).Error; err != nil {
		return nil
	}

	hash := &bc.Hash{}
	if err := hash.UnmarshalText([]byte(bsj.Hash)); err != nil {
		common.PanicCrisis(common.Fmt("Could not unmarshalText bytes: %s", bsj.Hash))
	}

	return &protocol.BlockStoreState{Height: bsj.Height, Hash: hash}
}

// A SQLStore encapsulates storage for blockchain validation.
// It satisfies the interface protocol.Store, and provides additional
// methods for querying current data.
type SQLStore struct {
	db    dbm.SQLDB
	cache blockCache
}

// GetBlockFromSQLDB return the block by given hash
func GetBlockFromSQLDB(db dbm.SQLDB, hash *bc.Hash) *types.Block {
	ormBlock := orm.Block{BlockHash: hash.String()}
	if err := db.Db().Where(&ormBlock).Find(&ormBlock).Error; err != nil {
		return nil
	}

	block := &types.Block{}
	block.UnmarshalText([]byte(ormBlock.Block))
	return block
}

// NewSQLStore creates and returns a new Store object.
func NewSQLStore(db dbm.SQLDB) *SQLStore {
	cache := newBlockCache(func(hash *bc.Hash) *types.Block {
		return GetBlockFromSQLDB(db, hash)
	})
	return &SQLStore{
		db:    db,
		cache: cache,
	}
}

// GetUtxo will search the utxo in db
func (s *SQLStore) GetUtxo(hash *bc.Hash) (*storage.UtxoEntry, error) {
	return getUtxoFromSQLDB(s.db, hash)
}

// BlockExist check if the block is stored in disk
func (s *SQLStore) BlockExist(hash *bc.Hash) bool {
	block, err := s.cache.lookup(hash)
	return err == nil && block != nil
}

// GetBlock return the block by given hash
func (s *SQLStore) GetBlock(hash *bc.Hash) (*types.Block, error) {
	return s.cache.lookup(hash)
}

// GetTransactionsUtxo will return all the utxo that related to the input txs
func (s *SQLStore) GetTransactionsUtxo(view *state.UtxoViewpoint, txs []*bc.Tx) error {
	return getTransactionsUtxoFromSQLDB(s.db, view, txs)
}

// GetTransactionStatus will return the utxo that related to the block hash
func (s *SQLStore) GetTransactionStatus(hash *bc.Hash) (*bc.TransactionStatus, error) {
	block := orm.Block{
		BlockHash: hash.String(),
	}
	if err := s.db.Db().Where(&block).Find(&block).Error; err != nil {
		return nil, err
	}

	ts := &bc.TransactionStatus{}
	if err := proto.Unmarshal([]byte(block.TxStatus), ts); err != nil {
		return nil, errors.Wrap(err, "unmarshaling transaction status")
	}
	return ts, nil
}

// GetStoreStatus return the BlockStoreStateJSON
func (s *SQLStore) GetStoreStatus() *protocol.BlockStoreState {
	return loadBlockSQLStoreStateJSON(s.db)
}

func (s *SQLStore) LoadBlockIndex(stateBestHeight uint64) (*state.BlockIndex, error) {
	startTime := time.Now()
	blockIndex := state.NewBlockIndex()
	start := uint64(0)
	limit := uint64(10000)
	var lastNode *state.BlockNode
loop:
	for {
		blocks := []orm.Block{}
		if err := s.db.Db().Offset(start).Limit(limit).Select("header").Find(&blocks).Error; err != nil {
			return nil, err
		}

		if len(blocks) == 0 {
			break loop
		}
		start += limit

		for _, block := range blocks {
			bh := &types.BlockHeader{}
			if err := bh.UnmarshalText([]byte(block.Header)); err != nil {
				return nil, err
			}

			if bh.Height > stateBestHeight {
				break loop
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

	}

	log.WithFields(log.Fields{
		"module":   logModule,
		"height":   stateBestHeight,
		"duration": time.Since(startTime),
	}).Debug("initialize load history block index from database")
	return blockIndex, nil
}

// SaveBlock persists a new block in the protocol.
func (s *SQLStore) SaveBlock(block *types.Block, ts *bc.TransactionStatus) error {
	startTime := time.Now()
	binaryBlock, err := block.MarshalText()
	if err != nil {
		return errors.Wrap(err, "Marshal block meta")
	}

	binaryBlockHeader, err := block.BlockHeader.MarshalText()
	if err != nil {
		return errors.Wrap(err, "Marshal block header")
	}

	binaryTxStatus, err := proto.Marshal(ts)
	if err != nil {
		return errors.Wrap(err, "marshal block transaction status")
	}

	blockHash := block.Hash()
	SQLDB := s.db.Db()
	b, err := blockHash.MarshalText()
	if err != nil {
		return err
	}
	blockInsert := &orm.Block{
		BlockHash: string(b),
		Height:    block.Height,
		Block:     string(binaryBlock),
		Header:    string(binaryBlockHeader),
		TxStatus:  string(binaryTxStatus),
	}
	if err := SQLDB.Save(blockInsert).Error; err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"module":   logModule,
		"height":   block.Height,
		"hash":     blockHash.String(),
		"duration": time.Since(startTime),
	}).Info("block saved on disk")
	return nil
}

// SaveChainStatus save the core's newest status && delete old status
func (s *SQLStore) SaveChainStatus(node *state.BlockNode, view *state.UtxoViewpoint) error {
	if err := saveUtxoViewToSQLDB(s.db, view); err != nil {
		return err
	}
	state := &orm.BlockStoreState{
		StoreKey: string(blockStoreKey),
		Height:   node.Height,
		Hash:     node.Hash.String(),
	}

	return s.db.Db().Save(state).Error
}

/*
func (s *SQLStore) IsWithdrawSpent(hash *bc.Hash) (bool, error) {
	data := &orm.ClaimTx{
		TxHash: hash.String(),
	}
	count := 0
	if err := s.db.Db().Where(data).First(data).Count(&count).Error; err != nil {
		return false, err
	}
	if count == 1 {
		return true, nil
	} else if count == 0 {
		return false, nil
	}

	return false, errors.New("Transactions of claim have duplicate records")
}
*/

func (s *SQLStore) IsWithdrawSpent(hash *bc.Hash) bool {
	data := &orm.ClaimTx{
		TxHash: hash.String(),
	}
	count := 0
	if err := s.db.Db().Where(data).First(data).Count(&count).Error; err != nil {
		return false
	}
	if count == 1 {
		return true
	} else if count == 0 {
		return false
	}

	return true
}

func (s *SQLStore) SetWithdrawSpent(hash *bc.Hash) error {
	data := &orm.ClaimTx{
		TxHash: hash.String(),
	}
	if err := s.db.Db().Save(data).Error; err != nil {
		return err
	}
	return nil
}
