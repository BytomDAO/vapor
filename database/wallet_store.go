package database

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	acc "github.com/bytom/vapor/account"
	"github.com/bytom/vapor/asset"
	"github.com/bytom/vapor/blockchain/query"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/wallet"
)

const (
	sutxoPrefix byte = iota //SUTXOPrefix is ContractUTXOKey prefix
	accountAliasPrefix
	txPrefix            //TxPrefix is wallet database transactions prefix
	txIndexPrefix       //TxIndexPrefix is wallet database tx index prefix
	unconfirmedTxPrefix //UnconfirmedTxPrefix is txpool unconfirmed transactions prefix
	globalTxIndexPrefix //GlobalTxIndexPrefix is wallet database global tx index prefix
	walletKey
	miningAddressKey
	coinbaseAbKey
	recoveryKey //recoveryKey key for db store recovery info.
)

// pre-define variables
var (
	walletStore         = []byte("WS:")
	SUTXOPrefix         = append(walletStore, sutxoPrefix, colon)
	AccountAliasPrefix  = append(walletStore, accountAliasPrefix, colon)
	TxPrefix            = append(walletStore, txPrefix, colon)            //TxPrefix is wallet database transactions prefix
	TxIndexPrefix       = append(walletStore, txIndexPrefix, colon)       //TxIndexPrefix is wallet database tx index prefix
	UnconfirmedTxPrefix = append(walletStore, unconfirmedTxPrefix, colon) //UnconfirmedTxPrefix is txpool unconfirmed transactions prefix
	GlobalTxIndexPrefix = append(walletStore, globalTxIndexPrefix, colon) //GlobalTxIndexPrefix is wallet database global tx index prefix
	WalletKey           = append(walletStore, walletKey)
	MiningAddressKey    = append(walletStore, miningAddressKey)
	CoinbaseAbKey       = append(walletStore, coinbaseAbKey)
	RecoveryKey         = append(walletStore, recoveryKey)
)

// ContractUTXOKey makes a smart contract unspent outputs key to store
func ContractUTXOKey(id bc.Hash) []byte {
	return append(SUTXOPrefix, id.Bytes()...)
}

func calcDeleteKey(blockHeight uint64) []byte {
	return []byte(fmt.Sprintf("%s%016x", TxPrefix, blockHeight))
}

func calcTxIndexKey(txID string) []byte {
	return append(TxIndexPrefix, []byte(txID)...)
}

func calcAnnotatedKey(formatKey string) []byte {
	return append(TxPrefix, []byte(formatKey)...)
}

func calcUnconfirmedTxKey(formatKey string) []byte {
	return append(UnconfirmedTxPrefix, []byte(formatKey)...)
}

// CalcGlobalTxIndexKey calculate tx hash index key
func CalcGlobalTxIndexKey(txID string) []byte {
	return append(GlobalTxIndexPrefix, []byte(txID)...)
}

// CalcGlobalTxIndex calcuate the block index + position index key
func CalcGlobalTxIndex(blockHash *bc.Hash, position uint64) []byte {
	txIdx := make([]byte, 40)
	copy(txIdx[:32], blockHash.Bytes())
	binary.BigEndian.PutUint64(txIdx[32:], position)
	return txIdx
}

func formatKey(blockHeight uint64, position uint32) string {
	return fmt.Sprintf("%016x%08x", blockHeight, position)
}

func contractIndexKey(accountID string) []byte {
	return append(ContractIndexPrefix, []byte(accountID)...)
}

// WalletStore store wallet using leveldb
type WalletStore struct {
	db    dbm.DB
	batch dbm.Batch
}

// NewWalletStore create new WalletStore struct
func NewWalletStore(db dbm.DB) *WalletStore {
	return &WalletStore{
		db:    db,
		batch: nil,
	}
}

// InitBatch initial new wallet store
func (store *WalletStore) InitBatch() wallet.WalletStore {
	newStore := NewWalletStore(store.db)
	newStore.batch = newStore.db.NewBatch()
	return newStore
}

// CommitBatch commit batch
func (store *WalletStore) CommitBatch() error {
	if store.batch == nil {
		return errors.New("walletStore commit fail, store batch is nil")
	}

	store.batch.Write()
	store.batch = nil
	return nil
}

// DeleteContractUTXO delete contract utxo by outputID
func (store *WalletStore) DeleteContractUTXO(outputID bc.Hash) {
	if store.batch == nil {
		store.db.Delete(ContractUTXOKey(outputID))
	} else {
		store.batch.Delete(ContractUTXOKey(outputID))
	}
}

// DeleteRecoveryStatus delete recovery status
func (store *WalletStore) DeleteRecoveryStatus() {
	if store.batch == nil {
		store.db.Delete(RecoveryKey)
	} else {
		store.batch.Delete(RecoveryKey)
	}
}

// DeleteTransactions delete transactions when orphan block rollback
func (store *WalletStore) DeleteTransactions(height uint64) {
	batch := store.db.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}
	txIter := store.db.IteratorPrefix(calcDeleteKey(height))
	defer txIter.Release()

	for txIter.Next() {
		tmpTx := new(query.AnnotatedTx)
		if err := json.Unmarshal(txIter.Value(), tmpTx); err == nil {
			batch.Delete(calcTxIndexKey(tmpTx.ID.String()))
		} else {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on DeleteTransactions.")
		}
		batch.Delete(txIter.Key())
	}
	if store.batch == nil {
		batch.Write()
	}
}

// DeleteUnconfirmedTransaction delete unconfirmed tx by txID
func (store *WalletStore) DeleteUnconfirmedTransaction(txID string) {
	if store.batch == nil {
		store.db.Delete(calcUnconfirmedTxKey(txID))
	} else {
		store.batch.Delete(calcUnconfirmedTxKey(txID))
	}
}

// DeleteWalletTransactions delete all txs in wallet
func (store *WalletStore) DeleteWalletTransactions() {
	batch := store.db.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}
	txIter := store.db.IteratorPrefix(TxPrefix)
	defer txIter.Release()

	for txIter.Next() {
		batch.Delete(txIter.Key())
	}

	txIndexIter := store.db.IteratorPrefix(TxIndexPrefix)
	defer txIndexIter.Release()

	for txIndexIter.Next() {
		batch.Delete(txIndexIter.Key())
	}
	if store.batch == nil {
		batch.Write()
	}
}

// DeleteWalletUTXOs delete all utxos in wallet
func (store *WalletStore) DeleteWalletUTXOs() {
	batch := store.db.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}

	ruIter := store.db.IteratorPrefix(UTXOPrefix)
	defer ruIter.Release()

	for ruIter.Next() {
		batch.Delete(ruIter.Key())
	}

	suIter := store.db.IteratorPrefix(SUTXOPrefix)
	defer suIter.Release()

	for suIter.Next() {
		batch.Delete(suIter.Key())
	}
	if store.batch == nil {
		batch.Write()
	}
}

// GetAsset get asset by assetID
func (store *WalletStore) GetAsset(assetID *bc.AssetID) (*asset.Asset, error) {
	definitionByte := store.db.Get(asset.ExtAssetKey(assetID))
	if definitionByte == nil {
		return nil, wallet.ErrGetAsset
	}

	definitionMap := make(map[string]interface{})
	if err := json.Unmarshal(definitionByte, &definitionMap); err != nil {
		return nil, err
	}

	alias := strings.ToUpper(assetID.String())
	externalAsset := &asset.Asset{
		AssetID:           *assetID,
		Alias:             &alias,
		DefinitionMap:     definitionMap,
		RawDefinitionByte: definitionByte,
	}
	return externalAsset, nil
}

// GetGlobalTransactionIndex get global tx by txID
func (store *WalletStore) GetGlobalTransactionIndex(txID string) []byte {
	return store.db.Get(CalcGlobalTxIndexKey(txID))
}

// GetStandardUTXO get standard utxo by id
func (store *WalletStore) GetStandardUTXO(outid bc.Hash) (*acc.UTXO, error) {
	rawUTXO := store.db.Get(StandardUTXOKey(outid))
	if rawUTXO == nil {
		return nil, wallet.ErrGetStandardUTXO
	}

	UTXO := new(acc.UTXO)
	if err := json.Unmarshal(rawUTXO, UTXO); err != nil {
		return nil, err
	}

	return UTXO, nil
}

// GetTransaction get tx by txid
func (store *WalletStore) GetTransaction(txID string) (*query.AnnotatedTx, error) {
	formatKey := store.db.Get(calcTxIndexKey(txID))
	if formatKey == nil {
		return nil, wallet.ErrAccntTxIDNotFound
	}

	rawTx := store.db.Get(calcAnnotatedKey(string(formatKey)))
	tx := new(query.AnnotatedTx)
	if err := json.Unmarshal(rawTx, tx); err != nil {
		return nil, err
	}

	return tx, nil
}

// GetUnconfirmedTransaction get unconfirmed tx by txID
func (store *WalletStore) GetUnconfirmedTransaction(txID string) (*query.AnnotatedTx, error) {
	rawUnconfirmedTx := store.db.Get(calcUnconfirmedTxKey(txID))
	if rawUnconfirmedTx == nil {
		return nil, fmt.Errorf("failed get unconfirmed tx, txID: %s ", txID)
	}

	tx := new(query.AnnotatedTx)
	if err := json.Unmarshal(rawUnconfirmedTx, tx); err != nil {
		return nil, err
	}

	return tx, nil
}

// GetRecoveryStatus delete recovery status
func (store *WalletStore) GetRecoveryStatus() (*wallet.RecoveryState, error) {
	rawStatus := store.db.Get(RecoveryKey)
	if rawStatus == nil {
		return nil, wallet.ErrGetRecoveryStatus
	}

	state := new(wallet.RecoveryState)
	if err := json.Unmarshal(rawStatus, state); err != nil {
		return nil, err
	}

	return state, nil
}

// GetWalletInfo get wallet information
func (store *WalletStore) GetWalletInfo() (*wallet.StatusInfo, error) {
	rawStatus := store.db.Get(WalletKey)
	if rawStatus == nil {
		return nil, wallet.ErrGetWalletStatusInfo
	}

	status := new(wallet.StatusInfo)
	if err := json.Unmarshal(rawStatus, status); err != nil {
		return nil, err
	}

	return status, nil
}

// ListAccountUTXOs get all account unspent outputs
func (store *WalletStore) ListAccountUTXOs(id string, isSmartContract bool) ([]*acc.UTXO, error) {
	prefix := UTXOPrefix
	if isSmartContract {
		prefix = SUTXOPrefix
	}

	idBytes, err := hex.DecodeString(id)
	if err != nil {
		return nil, err
	}

	accountUtxoIter := store.db.IteratorPrefix(append(prefix, idBytes...))
	defer accountUtxoIter.Release()

	confirmedUTXOs := []*acc.UTXO{}
	for accountUtxoIter.Next() {
		utxo := new(acc.UTXO)
		if err := json.Unmarshal(accountUtxoIter.Value(), utxo); err != nil {
			return nil, err
		}

		confirmedUTXOs = append(confirmedUTXOs, utxo)
	}

	return confirmedUTXOs, nil
}

// ListTransactions list tx by filter args
func (store *WalletStore) ListTransactions(accountID string, StartTxID string, count uint, unconfirmed bool) ([]*query.AnnotatedTx, error) {
	annotatedTxs := []*query.AnnotatedTx{}
	var startKey []byte
	preFix := TxPrefix

	if StartTxID != "" {
		if unconfirmed {
			startKey = calcUnconfirmedTxKey(StartTxID)
		} else {
			formatKey := store.db.Get(calcTxIndexKey(StartTxID))
			if formatKey == nil {
				return nil, wallet.ErrAccntTxIDNotFound
			}

			startKey = calcAnnotatedKey(string(formatKey))
		}
	}

	if unconfirmed {
		preFix = UnconfirmedTxPrefix
	}

	itr := store.db.IteratorPrefixWithStart(preFix, startKey, true)
	defer itr.Release()

	for txNum := count; itr.Next() && txNum > 0; {
		annotatedTx := new(query.AnnotatedTx)
		if err := json.Unmarshal(itr.Value(), annotatedTx); err != nil {
			return nil, err
		}

		if accountID == "" || wallet.FindTransactionsByAccount(annotatedTx, accountID) {
			annotatedTxs = append([]*query.AnnotatedTx{annotatedTx}, annotatedTxs...)
			txNum--
		}
	}

	return annotatedTxs, nil
}

// ListUnconfirmedTransactions get all unconfirmed txs
func (store *WalletStore) ListUnconfirmedTransactions() ([]*query.AnnotatedTx, error) {
	annotatedTxs := []*query.AnnotatedTx{}
	txIter := store.db.IteratorPrefix(UnconfirmedTxPrefix)
	defer txIter.Release()

	for txIter.Next() {
		annotatedTx := new(query.AnnotatedTx)
		if err := json.Unmarshal(txIter.Value(), annotatedTx); err != nil {
			return nil, err
		}

		annotatedTxs = append(annotatedTxs, annotatedTx)
	}
	return annotatedTxs, nil
}

// SetAssetDefinition set assetID and definition
func (store *WalletStore) SetAssetDefinition(assetID *bc.AssetID, definition []byte) {
	if store.batch == nil {
		store.db.Set(asset.ExtAssetKey(assetID), definition)
	} else {
		store.batch.Set(asset.ExtAssetKey(assetID), definition)
	}
}

// SetContractUTXO set standard utxo
func (store *WalletStore) SetContractUTXO(outputID bc.Hash, utxo *acc.UTXO) error {
	data, err := json.Marshal(utxo)
	if err != nil {
		return err
	}

	if store.batch == nil {
		store.db.Set(ContractUTXOKey(outputID), data)
	} else {
		store.batch.Set(ContractUTXOKey(outputID), data)
	}
	return nil
}

// SetGlobalTransactionIndex set global tx index by blockhash and position
func (store *WalletStore) SetGlobalTransactionIndex(globalTxID string, blockHash *bc.Hash, position uint64) {
	if store.batch == nil {
		store.db.Set(CalcGlobalTxIndexKey(globalTxID), CalcGlobalTxIndex(blockHash, position))
	} else {
		store.batch.Set(CalcGlobalTxIndexKey(globalTxID), CalcGlobalTxIndex(blockHash, position))
	}
}

// SetRecoveryStatus set recovery status
func (store *WalletStore) SetRecoveryStatus(recoveryState *wallet.RecoveryState) error {
	rawStatus, err := json.Marshal(recoveryState)
	if err != nil {
		return err
	}

	if store.batch == nil {
		store.db.Set(RecoveryKey, rawStatus)
	} else {
		store.batch.Set(RecoveryKey, rawStatus)
	}
	return nil
}

// SetTransaction set raw transaction by block height and tx position
func (store *WalletStore) SetTransaction(height uint64, tx *query.AnnotatedTx) error {
	batch := store.db.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}

	rawTx, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	batch.Set(calcAnnotatedKey(formatKey(height, tx.Position)), rawTx)
	batch.Set(calcTxIndexKey(tx.ID.String()), []byte(formatKey(height, tx.Position)))

	if store.batch == nil {
		batch.Write()
	}
	return nil
}

// SetUnconfirmedTransaction set unconfirmed tx by txID
func (store *WalletStore) SetUnconfirmedTransaction(txID string, tx *query.AnnotatedTx) error {
	rawTx, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	if store.batch == nil {
		store.db.Set(calcUnconfirmedTxKey(txID), rawTx)
	} else {
		store.batch.Set(calcUnconfirmedTxKey(txID), rawTx)
	}
	return nil
}

// SetWalletInfo get wallet information
func (store *WalletStore) SetWalletInfo(status *wallet.StatusInfo) error {
	rawWallet, err := json.Marshal(status)
	if err != nil {
		return err
	}

	if store.batch == nil {
		store.db.Set(WalletKey, rawWallet)
	} else {
		store.batch.Set(WalletKey, rawWallet)
	}
	return nil
}
