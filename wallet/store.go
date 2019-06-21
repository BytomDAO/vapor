package wallet

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
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
	GetTxIndexByTxID(string) []byte
	GetTxByTxIndex([]byte) []byte
	GetGlobalTxByTxID(string) []byte
	GetTransactions() ([]*query.AnnotatedTx, error)
	GetAllUnconfirmedTxs() ([]*query.AnnotatedTx, error)
	GetUnconfirmedTxByTxID(string) []byte
	SetUnconfirmedTx(string, []byte)
	DeleteStardardUTXOByOutputID(bc.Hash)
	DeleteContractUTXOByOutputID(bc.Hash)
	SetStandardUTXO(bc.Hash, []byte)
	SetContractUTXO(bc.Hash, []byte)
	GetWalletInfo() []byte
	SetWalletInfo([]byte)
	DeleteAllWalletTxs()
	DeleteAllWalletUTXOs()
	GetAccountUtxos(string, string, bool, bool, []*account.UTXO) []*account.UTXO
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

// GetTxIndexByTxID get tx index by txID
func (store *LevelDBStore) GetTxIndexByTxID(txID string) []byte {
	return store.DB.Get(calcTxIndexKey(txID))
}

// GetTxByTxIndex get tx by tx index
func (store *LevelDBStore) GetTxByTxIndex(txIndex []byte) []byte {
	return store.DB.Get(calcAnnotatedKey(string(txIndex)))
}

// GetGlobalTxByTxID get global tx by txID
func (store *LevelDBStore) GetGlobalTxByTxID(txID string) []byte {
	return store.DB.Get(calcGlobalTxIndexKey(txID))
}

// GetTransactions get all walletDB transactions
func (store *LevelDBStore) GetTransactions() ([]*query.AnnotatedTx, error) {
	annotatedTxs := []*query.AnnotatedTx{}

	txIter := store.DB.IteratorPrefix([]byte(TxPrefix))
	defer txIter.Release()
	for txIter.Next() {
		annotatedTx := &query.AnnotatedTx{}
		if err := json.Unmarshal(txIter.Value(), &annotatedTx); err != nil {
			return nil, err
		}
		annotatedTxs = append(annotatedTxs, annotatedTx)
	}

	return annotatedTxs, nil
}

// GetAllUnconfirmedTxs get all unconfirmed txs
func (store *LevelDBStore) GetAllUnconfirmedTxs() ([]*query.AnnotatedTx, error) {
	annotatedTxs := []*query.AnnotatedTx{}
	txIter := store.DB.IteratorPrefix([]byte(UnconfirmedTxPrefix))
	defer txIter.Release()

	for txIter.Next() {
		annotatedTx := &query.AnnotatedTx{}
		if err := json.Unmarshal(txIter.Value(), &annotatedTx); err != nil {
			return nil, err
		}
		annotatedTxs = append(annotatedTxs, annotatedTx)
	}
	return annotatedTxs, nil
}

// GetUnconfirmedTxByTxID get unconfirmed tx by txID
func (store *LevelDBStore) GetUnconfirmedTxByTxID(txID string) []byte {
	return store.DB.Get(calcUnconfirmedTxKey(txID))
}

// SetUnconfirmedTx set unconfirmed tx by txID
func (store *LevelDBStore) SetUnconfirmedTx(txID string, rawTx []byte) {
	store.DB.Set(calcUnconfirmedTxKey(txID), rawTx)
}

// DeleteStardardUTXOByOutputID delete stardard utxo by outputID
func (store *LevelDBStore) DeleteStardardUTXOByOutputID(outputID bc.Hash) {
	batch := store.DB.NewBatch()
	batch.Delete(account.StandardUTXOKey(outputID))
}

// DeleteContractUTXOByOutputID delete contract utxo by outputID
func (store *LevelDBStore) DeleteContractUTXOByOutputID(outputID bc.Hash) {
	batch := store.DB.NewBatch()
	batch.Delete(account.ContractUTXOKey(outputID))
}

// SetStandardUTXO set standard utxo
func (store *LevelDBStore) SetStandardUTXO(outputID bc.Hash, data []byte) {
	batch := store.DB.NewBatch()
	batch.Set(account.StandardUTXOKey(outputID), data)
}

// SetContractUTXO set standard utxo
func (store *LevelDBStore) SetContractUTXO(outputID bc.Hash, data []byte) {
	batch := store.DB.NewBatch()
	batch.Set(account.ContractUTXOKey(outputID), data)
}

// GetWalletInfo get wallet information
func (store *LevelDBStore) GetWalletInfo() []byte {
	return store.DB.Get(walletKey)
}

// SetWalletInfo get wallet information
func (store *LevelDBStore) SetWalletInfo(rawWallet []byte) {
	batch := store.DB.NewBatch()
	batch.Set(walletKey, rawWallet)
	batch.Write()
}

// DeleteAllWalletTxs delete all txs in wallet
func (store *LevelDBStore) DeleteAllWalletTxs() {
	storeBatch := store.DB.NewBatch()

	txIter := store.DB.IteratorPrefix([]byte(TxPrefix))
	defer txIter.Release()

	for txIter.Next() {
		storeBatch.Delete(txIter.Key())
	}

	txIndexIter := store.DB.IteratorPrefix([]byte(TxIndexPrefix))
	defer txIndexIter.Release()

	for txIndexIter.Next() {
		storeBatch.Delete(txIndexIter.Key())
	}

	storeBatch.Write()
}

// DeleteAllWalletUTXOs delete all txs in wallet
func (store *LevelDBStore) DeleteAllWalletUTXOs() {
	storeBatch := store.DB.NewBatch()
	ruIter := store.DB.IteratorPrefix([]byte(account.UTXOPreFix))
	defer ruIter.Release()
	for ruIter.Next() {
		storeBatch.Delete(ruIter.Key())
	}

	suIter := store.DB.IteratorPrefix([]byte(account.SUTXOPrefix))
	defer suIter.Release()
	for suIter.Next() {
		storeBatch.Delete(suIter.Key())
	}
	storeBatch.Write()
}

// GetAccountUtxos get all account unspent outputs
func (store *LevelDBStore) GetAccountUtxos(accountID, id string, isSmartContract, vote bool, accountUtxos []*account.UTXO) []*account.UTXO {
	prefix := account.UTXOPreFix
	if isSmartContract {
		prefix = account.SUTXOPrefix
	}
	accountUtxoIter := store.DB.IteratorPrefix([]byte(prefix + id))
	defer accountUtxoIter.Release()

	for accountUtxoIter.Next() {
		accountUtxo := &account.UTXO{}
		if err := json.Unmarshal(accountUtxoIter.Value(), accountUtxo); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Warn("GetAccountUtxos fail on unmarshal utxo")
			continue
		}

		if vote && accountUtxo.Vote == nil {
			continue
		}

		if accountID == accountUtxo.AccountID || accountID == "" {
			accountUtxos = append(accountUtxos, accountUtxo)
		}
	}
	newAccountUtxos := []*account.UTXO{}
	newAccountUtxos = accountUtxos
	return newAccountUtxos
}
