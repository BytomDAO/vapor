package database

import (
	"time"

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
	blockHeader := &orm.BlockHeader{BlockHash: hash.String()}
	if err := db.Db().Where(blockHeader).Find(blockHeader).Error; err != nil {
		return nil
	}

	txs := []*orm.Transaction{}
	if err := db.Db().Where(&orm.Transaction{BlockHash: hash.String()}).Find(&txs).Error; err != nil {
		return nil
	}

	block, err := toBlock(blockHeader, txs)
	if err != nil {
		return nil
	}

	return block
}

func toBlock(header *orm.BlockHeader, txs []*orm.Transaction) (*types.Block, error) {

	blockHeader, err := header.BcBlockHeader()
	if err != nil {
		return nil, err
	}

	var transactions []*types.Tx

	for _, tx := range txs {
		transaction, err := tx.UnmarshalText()
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}

	block := &types.Block{
		BlockHeader:  *blockHeader,
		Transactions: transactions,
	}

	return block, nil
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
	ts := &bc.TransactionStatus{}

	txs := []*orm.Transaction{}
	if err := s.db.Db().Where(&orm.Transaction{BlockHash: hash.String()}).Select("version,tx_index,status_fail").Find(&txs).Error; err != nil {
		return nil, nil
	}

	ts.VerifyStatus = make([]*bc.TxVerifyResult, len(txs))

	for _, tx := range txs {
		ts.Version = tx.Version
		ts.VerifyStatus[tx.TxIndex] = &bc.TxVerifyResult{tx.StatusFail}
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

	var lastNode *state.BlockNode
	rows, err := s.db.Db().Model(&orm.BlockHeader{}).Order("height").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		header := orm.BlockHeader{}
		if err := rows.Scan(&header.ID, &header.BlockHash, &header.Height, &header.Version, &header.PreviousBlockHash, &header.Timestamp, &header.TransactionsMerkleRoot, &header.TransactionStatusHash); err != nil {
			return nil, err
		}
		if header.Height > stateBestHeight {
			break
		}

		previousBlockHash, err := header.PreBlockHash()
		if err != nil {
			return nil, err
		}

		var parent *state.BlockNode
		if lastNode == nil || lastNode.Hash == *previousBlockHash {
			parent = lastNode
		} else {
			parent = blockIndex.GetNode(previousBlockHash)
		}

		bh, err := header.BcBlockHeader()
		if err != nil {
			return nil, err
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
func (s *SQLStore) SaveBlock(block *types.Block, ts *bc.TransactionStatus) error {
	startTime := time.Now()

	blockHash := block.Hash()
	SQLDB := s.db.Db()
	tx := SQLDB.Begin()
	// Save block header details
	blockHeader := &orm.BlockHeader{
		Height:                 block.Height,
		BlockHash:              blockHash.String(),
		Version:                block.Version,
		PreviousBlockHash:      block.PreviousBlockHash.String(),
		Timestamp:              block.Timestamp,
		TransactionsMerkleRoot: block.TransactionsMerkleRoot.String(),
		TransactionStatusHash:  block.TransactionStatusHash.String(),
	}

	if err := tx.Create(blockHeader).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Save tx
	for index, transaction := range block.Transactions {
		rawTx, err := transaction.MarshalText()
		if err != nil {
			return err
		}
		ormTransaction := &orm.Transaction{
			BlockHeaderID:  blockHeader.ID,
			BlockHash:      blockHash.String(),
			BlockHeight:    block.Height,
			Version:        block.Version,
			BlockTimestamp: block.Timestamp,
			TxIndex:        uint64(index),
			RawData:        string(rawTx),
			StatusFail:     ts.VerifyStatus[index].StatusFail,
		}
		if err := tx.Create(ormTransaction).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "commit transaction")
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

func (s *SQLStore) IsWithdrawSpent(hash *bc.Hash) bool {
	data := &orm.ClaimTxState{
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
	data := &orm.ClaimTxState{
		TxHash: hash.String(),
	}
	if err := s.db.Db().Save(data).Error; err != nil {
		return err
	}
	return nil
}
