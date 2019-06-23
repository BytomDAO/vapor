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
type WalletStorer interface {
	GetAssetDefinition(*bc.AssetID) []byte
	SetAssetDefinition(*bc.AssetID, []byte)
	GetRawProgram(common.Hash) []byte
	GetAccount(string) []byte
	DeleteTransaction(uint64)
	SetTransaction(uint64, uint32, string, []byte)
	DeleteUnconfirmedTransaction(string)
	SetGlobalTransactionIndex(string, *bc.Hash, uint64)
	GetStandardUTXO(bc.Hash) []byte
	GetTransaction(string) ([]byte, error)
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
	GetAccountUTXOs(key string) []*account.UTXO
	SetRecoveryStatus([]byte, []byte)
	DeleteRecoveryStatus([]byte)
	GetRecoveryStatus([]byte) []byte
}

// WalletStore store wallet using leveldb
type WalletStore struct {
	DB dbm.DB
}

// NewWalletStore create new WalletStore struct
func NewWalletStore(db dbm.DB) *WalletStore {
	return &WalletStore{
		DB: db,
	}
}

// GetAssetDefinition get asset definition by assetiD
func (store *WalletStore) GetAssetDefinition(assetID *bc.AssetID) []byte {
	return store.DB.Get(asset.ExtAssetKey(assetID))
}

// SetAssetDefinition set assetID and definition
func (store *WalletStore) SetAssetDefinition(assetID *bc.AssetID, definition []byte) {
	store.DB.Set(asset.ExtAssetKey(assetID), definition)
}

// GetRawProgram get raw program by hash
func (store *WalletStore) GetRawProgram(hash common.Hash) []byte {
	return store.DB.Get(account.ContractKey(hash))
}

// GetAccount get account value by account ID
func (store *WalletStore) GetAccount(accountID string) []byte {
	return store.DB.Get(account.Key(accountID))
}

// DeleteTransaction delete transactions when orphan block rollback
func (store *WalletStore) DeleteTransaction(height uint64) {
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

// SetTransaction set raw transaction by block height and tx position
func (store *WalletStore) SetTransaction(height uint64, position uint32, txID string, rawTx []byte) {
	batch := store.DB.NewBatch()
	batch.Set(calcAnnotatedKey(formatKey(height, position)), rawTx)
	batch.Set(calcTxIndexKey(txID), []byte(formatKey(height, position)))
	batch.Write()
}

// DeleteUnconfirmedTransaction delete unconfirmed tx by txID
func (store *WalletStore) DeleteUnconfirmedTransaction(txID string) {
	store.DB.Delete(calcUnconfirmedTxKey(txID))
}

// SetGlobalTransactionIndex set global tx index by blockhash and position
func (store *WalletStore) SetGlobalTransactionIndex(globalTxID string, blockHash *bc.Hash, position uint64) {
	store.DB.Set(calcGlobalTxIndexKey(globalTxID), calcGlobalTxIndex(blockHash, position))
}

// GetStandardUTXO get standard utxo by id
func (store *WalletStore) GetStandardUTXO(outid bc.Hash) []byte {
	return store.DB.Get(account.StandardUTXOKey(outid))
}

// GetTransaction get tx by tx index
func (store *WalletStore) GetTransaction(txID string) ([]byte, error) {
	formatKey := store.DB.Get(calcTxIndexKey(txID))
	if formatKey == nil {
		return nil, errAccntTxIDNotFound
	}
	txInfo := store.DB.Get(calcAnnotatedKey(string(formatKey)))
	return txInfo, nil
}

// GetGlobalTransaction get global tx by txID
func (store *WalletStore) GetGlobalTransaction(txID string) []byte {
	return store.DB.Get(calcGlobalTxIndexKey(txID))
}

// GetTransactions get all walletDB transactions
func (store *WalletStore) GetTransactions() ([]*query.AnnotatedTx, error) {
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
func (store *WalletStore) GetUnconfirmedTransactions() ([]*query.AnnotatedTx, error) {
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
func (store *WalletStore) GetUnconfirmedTransaction(txID string) []byte {
	return store.DB.Get(calcUnconfirmedTxKey(txID))
}

// SetUnconfirmedTransaction set unconfirmed tx by txID
func (store *WalletStore) SetUnconfirmedTransaction(txID string, rawTx []byte) {
	store.DB.Set(calcUnconfirmedTxKey(txID), rawTx)
}

// DeleteStardardUTXO delete stardard utxo by outputID
func (store *WalletStore) DeleteStardardUTXO(outputID bc.Hash) {
	store.DB.Delete(account.StandardUTXOKey(outputID))
}

// DeleteContractUTXO delete contract utxo by outputID
func (store *WalletStore) DeleteContractUTXO(outputID bc.Hash) {
	store.DB.Delete(account.ContractUTXOKey(outputID))
}

// SetStandardUTXO set standard utxo
func (store *WalletStore) SetStandardUTXO(outputID bc.Hash, data []byte) {
	store.DB.Set(account.StandardUTXOKey(outputID), data)
}

// SetContractUTXO set standard utxo
func (store *WalletStore) SetContractUTXO(outputID bc.Hash, data []byte) {
	store.DB.Set(account.ContractUTXOKey(outputID), data)
}

// GetWalletInfo get wallet information
func (store *WalletStore) GetWalletInfo() []byte {
	return store.DB.Get(walletKey)
}

// SetWalletInfo get wallet information
func (store *WalletStore) SetWalletInfo(rawWallet []byte) {
	store.DB.Set(walletKey, rawWallet)
}

// DeleteWalletTransactions delete all txs in wallet
func (store *WalletStore) DeleteWalletTransactions() {
	batch := store.DB.NewBatch()

	txIter := store.DB.IteratorPrefix([]byte(TxPrefix))
	defer txIter.Release()

	for txIter.Next() {
		batch.Delete(txIter.Key())
	}

	txIndexIter := store.DB.IteratorPrefix([]byte(TxIndexPrefix))
	defer txIndexIter.Release()

	for txIndexIter.Next() {
		batch.Delete(txIndexIter.Key())
	}

	batch.Write()
}

// DeleteWalletUTXOs delete all txs in wallet
func (store *WalletStore) DeleteWalletUTXOs() {
	batch := store.DB.NewBatch()
	ruIter := store.DB.IteratorPrefix([]byte(account.UTXOPreFix))
	defer ruIter.Release()
	for ruIter.Next() {
		batch.Delete(ruIter.Key())
	}

	suIter := store.DB.IteratorPrefix([]byte(account.SUTXOPrefix))
	defer suIter.Release()
	for suIter.Next() {
		batch.Delete(suIter.Key())
	}
	batch.Write()
}

// GetAccountUTXOs get all account unspent outputs
func (store *WalletStore) GetAccountUTXOs(key string) []*account.UTXO {
	accountUtxos := []*account.UTXO{}
	accountUtxoIter := store.DB.IteratorPrefix([]byte(key))
	defer accountUtxoIter.Release()

	for accountUtxoIter.Next() {
		accountUtxo := &account.UTXO{}
		if err := json.Unmarshal(accountUtxoIter.Value(), accountUtxo); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Warn("GetAccountUTXOs fail on unmarshal utxo")
			continue
		}
		accountUtxos = append(accountUtxos, accountUtxo)

	}
	return accountUtxos
}

// SetRecoveryStatus set recovery status
func (store *WalletStore) SetRecoveryStatus(recoveryKey, rawStatus []byte) {
	store.DB.Set(recoveryKey, rawStatus)
}

// DeleteRecoveryStatus delete recovery status
func (store *WalletStore) DeleteRecoveryStatus(recoveryKey []byte) {
	store.DB.Delete(recoveryKey)
}

// GetRecoveryStatus delete recovery status
func (store *WalletStore) GetRecoveryStatus(recoveryKey []byte) []byte {
	return store.DB.Get(recoveryKey)
}
