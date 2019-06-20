package wallet

import (
	"encoding/json"

	"github.com/vapor/account"
	"github.com/vapor/asset"
	"github.com/vapor/blockchain/query"
	"github.com/vapor/common"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

// DB interface contains wallet storage functions.
type DB interface {
	GetAssetDefinitionByAssetID(*bc.AssetID) []byte
	SetAssetDefinition(*bc.AssetID, []byte)
	GetRawProgramByAccountHash(common.Hash) []byte
	GetAccountValueByAccountID(string) []byte
	DeleteTransactionByHeight(uint64)
	SetRawTransaction(uint64, uint32, []byte)
	SaveExternalAssetDefinition(*types.Block)
	SetHeightAndPostion(string, uint64, uint32)
	DeleteUnconfirmedTxByTxID(string)
	SetGlobalTxIndex(string, *bc.Hash, uint64)
	GetStandardUTXOByID(bc.Hash) []byte
}

// LevelDBStore store wallet using leveldb
type LevelDBStore struct {
	DB dbm.DB
}

// NewLevelDBStore create new LevelDBStore struct
func NewLevelDBStore(db dbm.DB) *LevelDBStore {
	return &LevelDBStore{
		DB: db,
	}
}

// GetAssetDefinitionByAssetID get asset definition by assetiD
func (store *LevelDBStore) GetAssetDefinitionByAssetID(assetID *bc.AssetID) []byte {
	return store.DB.Get(asset.ExtAssetKey(assetID))
}

// SetAssetDefinition set assetID and definition
func (store *LevelDBStore) SetAssetDefinition(assetID *bc.AssetID, definition []byte) {
	batch := store.DB.NewBatch()
	batch.Set(asset.ExtAssetKey(assetID), definition)
}

// GetRawProgramByAccountHash get raw program by account hash
func (store *LevelDBStore) GetRawProgramByAccountHash(hash common.Hash) []byte {
	return store.DB.Get(account.ContractKey(hash))
}

// GetAccountValueByAccountID get account value by account ID
func (store *LevelDBStore) GetAccountValueByAccountID(accountID string) []byte {
	return store.DB.Get(account.Key(accountID))
}

// DeleteTransactionByHeight delete transactions when orphan block rollback
func (store *LevelDBStore) DeleteTransactionByHeight(height uint64) {
	tmpTx := query.AnnotatedTx{}
	batch := store.DB.NewBatch()
	txIter := store.DB.IteratorPrefix(calcDeleteKey(height))
	defer txIter.Release()

	for txIter.Next() {
		if err := json.Unmarshal(txIter.Value(), &tmpTx); err == nil {
			batch.Delete(calcTxIndexKey(tmpTx.ID.String()))
		}
		batch.Delete(txIter.Key())
	}
}

// SetRawTransaction set raw transaction by block height and tx position
func (store *LevelDBStore) SetRawTransaction(height uint64, position uint32, rawTx []byte) {
	batch := store.DB.NewBatch()
	batch.Set(calcAnnotatedKey(formatKey(height, position)), rawTx)
}

// SetHeightAndPostion set block height and tx position according to tx ID
func (store *LevelDBStore) SetHeightAndPostion(txID string, height uint64, position uint32) {
	batch := store.DB.NewBatch()
	batch.Set(calcTxIndexKey(txID), []byte(formatKey(height, position)))
}

// DeleteUnconfirmedTxByTxID delete unconfirmed tx by txID
func (store *LevelDBStore) DeleteUnconfirmedTxByTxID(txID string) {
	batch := store.DB.NewBatch()
	batch.Delete(calcUnconfirmedTxKey(txID))
}

// SetGlobalTxIndex set global tx index by blockhash and position
func (store *LevelDBStore) SetGlobalTxIndex(globalTxID string, blockHash *bc.Hash, position uint64) {
	batch := store.DB.NewBatch()
	batch.Set(calcGlobalTxIndexKey(globalTxID), calcGlobalTxIndex(blockHash, position))
}

// GetStandardUTXOByID get standard utxo by id
func (store *LevelDBStore) GetStandardUTXOByID(outid bc.Hash) []byte {
	return store.DB.Get(account.StandardUTXOKey(outid))
}
