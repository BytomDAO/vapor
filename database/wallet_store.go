package database

import (
	"encoding/json"

	"github.com/vapor/asset"
	"github.com/vapor/blockchain/query"
	"github.com/vapor/common"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
)

var errAccntTxIDNotFound = errors.New("account TXID not found")

// WalletStorer interface contains wallet storage functions.
type WalletStorer interface {
	InitBatch()
	CommitBatch()
	GetAssetDefinition(*bc.AssetID) []byte
	SetAssetDefinition(*bc.AssetID, []byte)
	GetRawProgram(common.Hash) []byte
	GetAccountByAccountID(string) []byte
	DeleteTransactions(uint64)
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
	GetAccountUTXOs(key string) [][]byte
	SetRecoveryStatus([]byte, []byte)
	DeleteRecoveryStatus([]byte)
	GetRecoveryStatus([]byte) []byte
}

// WalletStore store wallet using leveldb
type WalletStore struct {
	walletDB dbm.DB
	batch    dbm.Batch
}

// NewWalletStore create new WalletStore struct
func NewWalletStore(db dbm.DB) *WalletStore {
	return &WalletStore{
		walletDB: db,
		batch:    nil,
	}
}

// InitBatch initial batch
func (store *WalletStore) InitBatch() {
	if store.batch == nil {
		store.batch = store.walletDB.NewBatch()
	}
}

// CommitBatch commit batch
func (store *WalletStore) CommitBatch() {
	if store.batch != nil {
		store.batch.Write()
		store.batch = nil
	}
}

// GetAssetDefinition get asset definition by assetiD
func (store *WalletStore) GetAssetDefinition(assetID *bc.AssetID) []byte {
	return store.walletDB.Get(asset.ExtAssetKey(assetID))
}

// SetAssetDefinition set assetID and definition
func (store *WalletStore) SetAssetDefinition(assetID *bc.AssetID, definition []byte) {
	if store.batch == nil {
		store.walletDB.Set(asset.ExtAssetKey(assetID), definition)
	} else {
		store.batch.Set(asset.ExtAssetKey(assetID), definition)
	}
}

// GetRawProgram get raw program by hash
func (store *WalletStore) GetRawProgram(hash common.Hash) []byte {
	return store.walletDB.Get(ContractKey(hash))
}

// GetAccountByAccountID get account value by account ID
func (store *WalletStore) GetAccountByAccountID(accountID string) []byte {
	return store.walletDB.Get(AccountIDKey(accountID))
}

// DeleteTransactions delete transactions when orphan block rollback
func (store *WalletStore) DeleteTransactions(height uint64) {
	tmpTx := query.AnnotatedTx{}
	batch := store.walletDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}
	txIter := store.walletDB.IteratorPrefix(calcDeleteKey(height))
	defer txIter.Release()

	for txIter.Next() {
		if err := json.Unmarshal(txIter.Value(), &tmpTx); err == nil {
			batch.Delete(calcTxIndexKey(tmpTx.ID.String()))
		}
		batch.Delete(txIter.Key())
	}
	if store.batch == nil {
		batch.Write()
	}
}

// SetTransaction set raw transaction by block height and tx position
func (store *WalletStore) SetTransaction(height uint64, position uint32, txID string, rawTx []byte) {
	batch := store.walletDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}

	batch.Set(calcAnnotatedKey(formatKey(height, position)), rawTx)
	batch.Set(calcTxIndexKey(txID), []byte(formatKey(height, position)))

	if store.batch == nil {
		batch.Write()
	}
}

// DeleteUnconfirmedTransaction delete unconfirmed tx by txID
func (store *WalletStore) DeleteUnconfirmedTransaction(txID string) {
	if store.batch == nil {
		store.walletDB.Delete(calcUnconfirmedTxKey(txID))
	} else {
		store.batch.Delete(calcUnconfirmedTxKey(txID))
	}
}

// SetGlobalTransactionIndex set global tx index by blockhash and position
func (store *WalletStore) SetGlobalTransactionIndex(globalTxID string, blockHash *bc.Hash, position uint64) {
	if store.batch == nil {
		store.walletDB.Set(calcGlobalTxIndexKey(globalTxID), CalcGlobalTxIndex(blockHash, position))
	} else {
		store.batch.Set(calcGlobalTxIndexKey(globalTxID), CalcGlobalTxIndex(blockHash, position))
	}
}

// GetStandardUTXO get standard utxo by id
func (store *WalletStore) GetStandardUTXO(outid bc.Hash) []byte {
	return store.walletDB.Get(StandardUTXOKey(outid))
}

// GetTransaction get tx by tx index
func (store *WalletStore) GetTransaction(txID string) ([]byte, error) {
	formatKey := store.walletDB.Get(calcTxIndexKey(txID))
	if formatKey == nil {
		return nil, errAccntTxIDNotFound
	}
	txInfo := store.walletDB.Get(calcAnnotatedKey(string(formatKey)))
	return txInfo, nil
}

// GetGlobalTransaction get global tx by txID
func (store *WalletStore) GetGlobalTransaction(txID string) []byte {
	return store.walletDB.Get(calcGlobalTxIndexKey(txID))
}

// GetTransactions get all walletDB transactions
func (store *WalletStore) GetTransactions() ([]*query.AnnotatedTx, error) {
	annotatedTxs := []*query.AnnotatedTx{}

	txIter := store.walletDB.IteratorPrefix([]byte(TxPrefix))
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
	txIter := store.walletDB.IteratorPrefix([]byte(UnconfirmedTxPrefix))
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
	return store.walletDB.Get(calcUnconfirmedTxKey(txID))
}

// SetUnconfirmedTransaction set unconfirmed tx by txID
func (store *WalletStore) SetUnconfirmedTransaction(txID string, rawTx []byte) {
	if store.batch == nil {
		store.walletDB.Set(calcUnconfirmedTxKey(txID), rawTx)
	} else {
		store.batch.Set(calcUnconfirmedTxKey(txID), rawTx)
	}
}

// DeleteStardardUTXO delete stardard utxo by outputID
func (store *WalletStore) DeleteStardardUTXO(outputID bc.Hash) {
	if store.batch == nil {
		store.walletDB.Delete(StandardUTXOKey(outputID))
	} else {
		store.batch.Delete(StandardUTXOKey(outputID))
	}
}

// DeleteContractUTXO delete contract utxo by outputID
func (store *WalletStore) DeleteContractUTXO(outputID bc.Hash) {
	if store.batch == nil {
		store.walletDB.Delete(ContractUTXOKey(outputID))
	} else {
		store.batch.Delete(ContractUTXOKey(outputID))
	}
}

// SetStandardUTXO set standard utxo
func (store *WalletStore) SetStandardUTXO(outputID bc.Hash, data []byte) {
	if store.batch == nil {
		store.walletDB.Set(StandardUTXOKey(outputID), data)
	} else {
		store.batch.Set(StandardUTXOKey(outputID), data)
	}
}

// SetContractUTXO set standard utxo
func (store *WalletStore) SetContractUTXO(outputID bc.Hash, data []byte) {
	if store.batch == nil {
		store.walletDB.Set(ContractUTXOKey(outputID), data)
	} else {
		store.batch.Set(ContractUTXOKey(outputID), data)
	}
}

// GetWalletInfo get wallet information
func (store *WalletStore) GetWalletInfo() []byte {
	return store.walletDB.Get([]byte(WalletKey))
}

// SetWalletInfo get wallet information
func (store *WalletStore) SetWalletInfo(rawWallet []byte) {
	if store.batch == nil {
		store.walletDB.Set([]byte(WalletKey), rawWallet)
	} else {
		store.batch.Set([]byte(WalletKey), rawWallet)
	}
}

// DeleteWalletTransactions delete all txs in wallet
func (store *WalletStore) DeleteWalletTransactions() {
	batch := store.walletDB.NewBatch()
	txIter := store.walletDB.IteratorPrefix([]byte(TxPrefix))
	defer txIter.Release()

	for txIter.Next() {
		batch.Delete(txIter.Key())
	}

	txIndexIter := store.walletDB.IteratorPrefix([]byte(TxIndexPrefix))
	defer txIndexIter.Release()

	for txIndexIter.Next() {
		batch.Delete(txIndexIter.Key())
	}
	batch.Write()
}

// DeleteWalletUTXOs delete all txs in wallet
func (store *WalletStore) DeleteWalletUTXOs() {
	batch := store.walletDB.NewBatch()
	ruIter := store.walletDB.IteratorPrefix([]byte(UTXOPrefix))
	defer ruIter.Release()
	for ruIter.Next() {
		batch.Delete(ruIter.Key())
	}

	suIter := store.walletDB.IteratorPrefix([]byte(SUTXOPrefix))
	defer suIter.Release()
	for suIter.Next() {
		batch.Delete(suIter.Key())
	}
	batch.Write()
}

// GetAccountUTXOs get all account unspent outputs
func (store *WalletStore) GetAccountUTXOs(key string) [][]byte {
	accountUtxoIter := store.walletDB.IteratorPrefix([]byte(key))
	defer accountUtxoIter.Release()

	rawUTXOs := make([][]byte, 0)
	for accountUtxoIter.Next() {
		utxo := accountUtxoIter.Value()
		rawUTXOs = append(rawUTXOs, utxo)
	}
	return rawUTXOs
}

// SetRecoveryStatus set recovery status
func (store *WalletStore) SetRecoveryStatus(recoveryKey, rawStatus []byte) {
	if store.batch == nil {
		store.walletDB.Set(recoveryKey, rawStatus)
	} else {
		store.batch.Set(recoveryKey, rawStatus)
	}
}

// DeleteRecoveryStatus delete recovery status
func (store *WalletStore) DeleteRecoveryStatus(recoveryKey []byte) {
	if store.batch == nil {
		store.walletDB.Delete(recoveryKey)
	} else {
		store.batch.Delete(recoveryKey)
	}
}

// GetRecoveryStatus delete recovery status
func (store *WalletStore) GetRecoveryStatus(recoveryKey []byte) []byte {
	return store.walletDB.Get(recoveryKey)
}
