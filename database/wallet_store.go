package database

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"

	acc "github.com/vapor/account"
	"github.com/vapor/asset"
	"github.com/vapor/blockchain/query"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/wallet"
)

const (
	utxoPrefix  byte = iota //UTXOPrefix is StandardUTXOKey prefix
	sutxoPrefix             //SUTXOPrefix is ContractUTXOKey prefix
	contractPrefix
	contractIndexPrefix
	accountPrefix // AccountPrefix is account ID prefix
	accountAliasPrefix
	accountIndexPrefix
	txPrefix            //TxPrefix is wallet database transactions prefix
	txIndexPrefix       //TxIndexPrefix is wallet database tx index prefix
	unconfirmedTxPrefix //UnconfirmedTxPrefix is txpool unconfirmed transactions prefix
	globalTxIndexPrefix //GlobalTxIndexPrefix is wallet database global tx index prefix
	walletKey
	miningAddressKey
	coinbaseAbKey
	recoveryKey //recoveryKey key for db store recovery info.
)

// leveldb key prefix
var (
	// colon               byte = 0x3a
	UTXOPrefix          = []byte{utxoPrefix, colon}
	SUTXOPrefix         = []byte{sutxoPrefix, colon}
	ContractPrefix      = []byte{contractPrefix, contractPrefix, colon}
	ContractIndexPrefix = []byte{contractIndexPrefix, colon}
	AccountPrefix       = []byte{accountPrefix, colon} // AccountPrefix is account ID prefix
	AccountAliasPrefix  = []byte{accountAliasPrefix, colon}
	AccountIndexPrefix  = []byte{accountIndexPrefix, colon}
	TxPrefix            = []byte{txPrefix, colon}            //TxPrefix is wallet database transactions prefix
	TxIndexPrefix       = []byte{txIndexPrefix, colon}       //TxIndexPrefix is wallet database tx index prefix
	UnconfirmedTxPrefix = []byte{unconfirmedTxPrefix, colon} //UnconfirmedTxPrefix is txpool unconfirmed transactions prefix
	GlobalTxIndexPrefix = []byte{globalTxIndexPrefix, colon} //GlobalTxIndexPrefix is wallet database global tx index prefix
	WalletKey           = []byte{walletKey}
	MiningAddressKey    = []byte{miningAddressKey}
	CoinbaseAbKey       = []byte{coinbaseAbKey}
	RecoveryKey         = []byte{recoveryKey}
)

// errors
var (
	errAccntTxIDNotFound = errors.New("account TXID not found")
	errGetAsset          = errors.New("Failed to find asset definition")
)

func Bip44ContractIndexKey(accountID string, change bool) []byte {
	key := append(ContractIndexPrefix, []byte(accountID)...)
	if change {
		return append(key, []byte{1}...)
	}
	return append(key, []byte{0}...)
}

// ContractKey account control promgram store prefix
func ContractKey(hash bc.Hash) []byte {
	return append(ContractPrefix, hash.Bytes()...)
}

// AccountIDKey account id store prefix
func AccountIDKey(accountID string) []byte {
	return append(AccountPrefix, []byte(accountID)...)
}

// StandardUTXOKey makes an account unspent outputs key to store
func StandardUTXOKey(id bc.Hash) []byte {
	return append(UTXOPrefix, id.Bytes()...)
}

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

func CalcGlobalTxIndexKey(txID string) []byte {
	return append(GlobalTxIndexPrefix, []byte(txID)...)
}

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

func accountAliasKey(name string) []byte {
	return append(AccountAliasPrefix, []byte(name)...)
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

// InitBatch initial batch
func (store *WalletStore) InitBatch() error {
	if store.batch != nil {
		return errors.New("WalletStore initail fail, store batch is not nil.")
	}

	store.batch = store.db.NewBatch()
	return nil
}

// CommitBatch commit batch
func (store *WalletStore) CommitBatch() error {
	if store.batch == nil {
		return errors.New("WalletStore commit fail, store batch is nil.")
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

	tmpTx := query.AnnotatedTx{}
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

// DeleteWalletUTXOs delete all txs in wallet
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
		return nil, errGetAsset
	}

	definitionMap := make(map[string]interface{})
	if err := json.Unmarshal(definitionByte, &definitionMap); err != nil {
		return nil, err
	}

	alias := assetID.String()
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
		return nil, fmt.Errorf("failed get standard UTXO, outputID: %s ", outid.String())
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
		return nil, errAccntTxIDNotFound
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
				return nil, errAccntTxIDNotFound
			}

			startKey = calcAnnotatedKey(string(formatKey))
		}
	}

	if unconfirmed {
		preFix = UnconfirmedTxPrefix
	}

	itr := store.db.IteratorPrefixWithStart(preFix, startKey, true)
	defer itr.Release()

	for txNum := count; itr.Next() && txNum > 0; txNum-- {
		annotatedTx := new(query.AnnotatedTx)
		if err := json.Unmarshal(itr.Value(), &annotatedTx); err != nil {
			return nil, err
		}

		annotatedTxs = append(annotatedTxs, annotatedTx)
	}

	return annotatedTxs, nil
}

// ListUnconfirmedTransactions get all unconfirmed txs
func (store *WalletStore) ListUnconfirmedTransactions() ([]*query.AnnotatedTx, error) {
	annotatedTxs := []*query.AnnotatedTx{}
	txIter := store.db.IteratorPrefix(UnconfirmedTxPrefix)
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
