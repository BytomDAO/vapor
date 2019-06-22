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
)

// Store interface contains wallet storage functions.
type Store interface {
	GetAssetDefinition(*bc.AssetID) []byte
	SetAssetDefinition(*bc.AssetID, []byte)
	GetRawProgramByHash(common.Hash) []byte
	GetAccount(string) []byte
	DeleteTransaction(uint64)
	SetRawTransaction(uint64, uint32, []byte)
	SetHeightAndPostion(string, uint64, uint32)
	DeleteUnconfirmedTx(string)
	SetGlobalTxIndex(string, *bc.Hash, uint64)
	GetStandardUTXO(bc.Hash) []byte
	GetTransactionIndex(string) []byte
	GetTransaction([]byte) []byte
	GetGlobalTransaction(string) []byte
	GetTransactions() ([]*query.AnnotatedTx, error)
	GetUnconfirmedTransactions() ([]*query.AnnotatedTx, error)
	GetUnconfirmedTransaction(string) []byte
	SetUnconfirmedTransaction(string, []byte)
	DeleteStardardUTXO(bc.Hash)
	DeleteContractUTXO(bc.Hash)
	SetStandardUTXO(bc.Hash, []byte)
	SetContractUTXO(bc.Hash, []byte)
	GetWalletInfo() []byte
	SetWalletInfo([]byte)
	DeleteWalletTransactions()
	DeleteWalletUTXOs()
	GetAccountUtxos(key string) []*account.UTXO
	SetRecoveryStatus([]byte, []byte)
	DeleteRecoveryStatus([]byte)
	GetRecoveryStatus([]byte) []byte
}

// LevelDBStore store wallet using leveldb
type LevelDBStore struct {
	DB dbm.DB
}

// NewLevelDBStore create new LevelDBStore struct
func NewStore(db dbm.DB) *LevelDBStore {
	return &LevelDBStore{
		DB: db,
	}
}

// GetAssetDefinition get asset definition by assetiD
func (store *LevelDBStore) GetAssetDefinition(assetID *bc.AssetID) []byte {
	return store.DB.Get(asset.ExtAssetKey(assetID))
}

// SetAssetDefinition set assetID and definition
func (store *LevelDBStore) SetAssetDefinition(assetID *bc.AssetID, definition []byte) {
	batch := store.DB.NewBatch()
	batch.Set(asset.ExtAssetKey(assetID), definition)
	batch.Write()
}

// GetRawProgramByHash get raw program by hash
func (store *LevelDBStore) GetRawProgramByHash(hash common.Hash) []byte {
	return store.DB.Get(account.ContractKey(hash))
}

// GetAccount get account value by account ID
func (store *LevelDBStore) GetAccount(accountID string) []byte {
	return store.DB.Get(account.Key(accountID))
}

// DeleteTransaction delete transactions when orphan block rollback
func (store *LevelDBStore) DeleteTransaction(height uint64) {
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
	batch.Write()
}

// SetRawTransaction set raw transaction by block height and tx position
func (store *LevelDBStore) SetRawTransaction(height uint64, position uint32, rawTx []byte) {
	batch := store.DB.NewBatch()
	batch.Set(calcAnnotatedKey(formatKey(height, position)), rawTx)
	batch.Write()
}

// SetHeightAndPostion set block height and tx position according to tx ID
func (store *LevelDBStore) SetHeightAndPostion(txID string, height uint64, position uint32) {
	batch := store.DB.NewBatch()
	batch.Set(calcTxIndexKey(txID), []byte(formatKey(height, position)))
	batch.Write()
}

// DeleteUnconfirmedTx delete unconfirmed tx by txID
func (store *LevelDBStore) DeleteUnconfirmedTx(txID string) {
	batch := store.DB.NewBatch()
	batch.Delete(calcUnconfirmedTxKey(txID))
	batch.Write()
}

// SetGlobalTxIndex set global tx index by blockhash and position
func (store *LevelDBStore) SetGlobalTxIndex(globalTxID string, blockHash *bc.Hash, position uint64) {
	batch := store.DB.NewBatch()
	batch.Set(calcGlobalTxIndexKey(globalTxID), calcGlobalTxIndex(blockHash, position))
	batch.Write()
}

// GetStandardUTXO get standard utxo by id
func (store *LevelDBStore) GetStandardUTXO(outid bc.Hash) []byte {
	return store.DB.Get(account.StandardUTXOKey(outid))
}

// GetTransactionIndex get tx index by txID
func (store *LevelDBStore) GetTransactionIndex(txID string) []byte {
	return store.DB.Get(calcTxIndexKey(txID))
}

// GetTransaction get tx by tx index
func (store *LevelDBStore) GetTransaction(txIndex []byte) []byte {
	return store.DB.Get(calcAnnotatedKey(string(txIndex)))
}

// GetGlobalTransaction get global tx by txID
func (store *LevelDBStore) GetGlobalTransaction(txID string) []byte {
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

// GetUnconfirmedTransactions get all unconfirmed txs
func (store *LevelDBStore) GetUnconfirmedTransactions() ([]*query.AnnotatedTx, error) {
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

// GetUnconfirmedTransaction get unconfirmed tx by txID
func (store *LevelDBStore) GetUnconfirmedTransaction(txID string) []byte {
	return store.DB.Get(calcUnconfirmedTxKey(txID))
}

// SetUnconfirmedTransaction set unconfirmed tx by txID
func (store *LevelDBStore) SetUnconfirmedTransaction(txID string, rawTx []byte) {
	store.DB.Set(calcUnconfirmedTxKey(txID), rawTx)
}

// DeleteStardardUTXO delete stardard utxo by outputID
func (store *LevelDBStore) DeleteStardardUTXO(outputID bc.Hash) {
	batch := store.DB.NewBatch()
	batch.Delete(account.StandardUTXOKey(outputID))
	batch.Write()
}

// DeleteContractUTXO delete contract utxo by outputID
func (store *LevelDBStore) DeleteContractUTXO(outputID bc.Hash) {
	batch := store.DB.NewBatch()
	batch.Delete(account.ContractUTXOKey(outputID))
	batch.Write()
}

// SetStandardUTXO set standard utxo
func (store *LevelDBStore) SetStandardUTXO(outputID bc.Hash, data []byte) {
	batch := store.DB.NewBatch()
	batch.Set(account.StandardUTXOKey(outputID), data)
	batch.Write()
}

// SetContractUTXO set standard utxo
func (store *LevelDBStore) SetContractUTXO(outputID bc.Hash, data []byte) {
	batch := store.DB.NewBatch()
	batch.Set(account.ContractUTXOKey(outputID), data)
	batch.Write()
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

// DeleteWalletTransactions delete all txs in wallet
func (store *LevelDBStore) DeleteWalletTransactions() {
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

// DeleteWalletUTXOs delete all txs in wallet
func (store *LevelDBStore) DeleteWalletUTXOs() {
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
func (store *LevelDBStore) GetAccountUtxos(key string) []*account.UTXO {
	accountUtxos := []*account.UTXO{}
	accountUtxoIter := store.DB.IteratorPrefix([]byte(key))
	defer accountUtxoIter.Release()

	for accountUtxoIter.Next() {
		accountUtxo := &account.UTXO{}
		if err := json.Unmarshal(accountUtxoIter.Value(), accountUtxo); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Warn("GetAccountUtxos fail on unmarshal utxo")
			continue
		}
		accountUtxos = append(accountUtxos, accountUtxo)

	}
	return accountUtxos
}

// SetRecoveryStatus set recovery status
func (store *LevelDBStore) SetRecoveryStatus(recoveryKey, rawStatus []byte) {
	store.DB.Set(recoveryKey, rawStatus)
}

// DeleteRecoveryStatus delete recovery status
func (store *LevelDBStore) DeleteRecoveryStatus(recoveryKey []byte) {
	store.DB.Delete(recoveryKey)
}

// GetRecoveryStatus delete recovery status
func (store *LevelDBStore) GetRecoveryStatus(recoveryKey []byte) []byte {
	return store.DB.Get(recoveryKey)
}
