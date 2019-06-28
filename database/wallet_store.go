package database

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"

	acc "github.com/vapor/account"
	"github.com/vapor/asset"
	"github.com/vapor/blockchain/query"
	"github.com/vapor/blockchain/signers"
	"github.com/vapor/common"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/crypto/sha3pool"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
)

const (
	// UTXOPrefix          = "ACU:" //UTXOPrefix is StandardUTXOKey prefix
	// SUTXOPrefix         = "SCU:" //SUTXOPrefix is ContractUTXOKey prefix

	ContractPrefix = "Contract:"

// ContractIndexPrefix = "ContractIndex:"
// AccountPrefix       = "Account:" // AccountPrefix is account ID prefix
// AccountAliasPrefix  = "AccountAlias:"
// AccountIndexPrefix  = "AccountIndex:"
// TxPrefix            = "TXS:"  //TxPrefix is wallet database transactions prefix
// TxIndexPrefix       = "TID:"  //TxIndexPrefix is wallet database tx index prefix
// UnconfirmedTxPrefix = "UTXS:" //UnconfirmedTxPrefix is txpool unconfirmed transactions prefix
// GlobalTxIndexPrefix = "GTID:" //GlobalTxIndexPrefix is wallet database global tx index prefix
// WalletKey        = "WalletInfo"
// MiningAddressKey = "MiningAddress"
// CoinbaseAbKey    = "CoinbaseArbitrary"
)

const (
	utxoPrefix  byte = iota //UTXOPrefix is StandardUTXOKey prefix
	sUTXOPrefix             //SUTXOPrefix is ContractUTXOKey prefix
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
)

// leveldb key prefix
var (
	UTXOPrefix  = []byte{utxoPrefix, colon}
	SUTXOPrefix = []byte{sUTXOPrefix, colon}
	// ContractPrefix      = []byte{contractPrefix, colon}
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
)

// errors
var (
	// ErrFindAccount        = errors.New("Failed to find account")
	errAccntTxIDNotFound  = errors.New("account TXID not found")
	errGetAssetDefinition = errors.New("Failed to find asset definition")
)

func accountIndexKey(xpubs []chainkd.XPub) []byte {
	var hash [32]byte
	var xPubs []byte
	cpy := append([]chainkd.XPub{}, xpubs[:]...)
	sort.Sort(signers.SortKeys(cpy))
	for _, xpub := range cpy {
		xPubs = append(xPubs, xpub[:]...)
	}
	sha3pool.Sum256(hash[:], xPubs)
	return append([]byte(AccountIndexPrefix), hash[:]...)
}

func Bip44ContractIndexKey(accountID string, change bool) []byte {
	key := append([]byte(ContractIndexPrefix), accountID...)
	if change {
		return append(key, []byte{1}...)
	}
	return append(key, []byte{0}...)
}

// ContractKey account control promgram store prefix
func ContractKey(hash common.Hash) []byte {
	// h := hash.Str()
	// return append([]byte(ContractPrefix), []byte(h)...)
	return append([]byte(ContractPrefix), hash.Bytes()...)
}

// AccountIDKey account id store prefix
func AccountIDKey(accountID string) []byte {
	return append([]byte(AccountPrefix), []byte(accountID)...)
}

// StandardUTXOKey makes an account unspent outputs key to store
func StandardUTXOKey(id bc.Hash) []byte {
	name := id.String()
	return append(UTXOPrefix, []byte(name)...)
}

// ContractUTXOKey makes a smart contract unspent outputs key to store
func ContractUTXOKey(id bc.Hash) []byte {
	name := id.String()
	return append(SUTXOPrefix, []byte(name)...)
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

func calcGlobalTxIndexKey(txID string) []byte {
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
	return append([]byte(ContractIndexPrefix), []byte(accountID)...)
}

func accountAliasKey(name string) []byte {
	return append([]byte(AccountAliasPrefix), []byte(name)...)
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
func (store *WalletStore) GetAssetDefinition(assetID *bc.AssetID) (*asset.Asset, error) {
	definitionByte := store.walletDB.Get(asset.ExtAssetKey(assetID))
	if definitionByte == nil {
		return nil, errGetAssetDefinition
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

// SetAssetDefinition set assetID and definition
func (store *WalletStore) SetAssetDefinition(assetID *bc.AssetID, definition []byte) {
	if store.batch == nil {
		store.walletDB.Set(asset.ExtAssetKey(assetID), definition)
	} else {
		store.batch.Set(asset.ExtAssetKey(assetID), definition)
	}
}

// GetControlProgram get raw program by hash
func (store *WalletStore) GetControlProgram(hash common.Hash) (*acc.CtrlProgram, error) {
	rawProgram := store.walletDB.Get(ContractKey(hash))
	if rawProgram == nil {
		return nil, fmt.Errorf("failed get account control program:%x ", hash)
	}
	accountCP := new(acc.CtrlProgram)
	if err := json.Unmarshal(rawProgram, &accountCP); err != nil {
		return nil, err
	}
	return accountCP, nil
}

// GetAccountByAccountID get account value by account ID
func (store *WalletStore) GetAccountByAccountID(accountID string) (*acc.Account, error) {
	rawAccount := store.walletDB.Get(AccountIDKey(accountID))
	if rawAccount == nil {
		return nil, fmt.Errorf("failed get account, accountID: %s ", accountID)
	}
	account := new(acc.Account)
	if err := json.Unmarshal(rawAccount, account); err != nil {
		return nil, err
	}
	return account, nil
}

// DeleteTransactions delete transactions when orphan block rollback
func (store *WalletStore) DeleteTransactions(height uint64) {
	batch := store.walletDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}
	txIter := store.walletDB.IteratorPrefix(calcDeleteKey(height))
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

// SetTransaction set raw transaction by block height and tx position
func (store *WalletStore) SetTransaction(height uint64, tx *query.AnnotatedTx) error {
	batch := store.walletDB.NewBatch()
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
func (store *WalletStore) GetStandardUTXO(outid bc.Hash) (*acc.UTXO, error) {
	rawUTXO := store.walletDB.Get(StandardUTXOKey(outid))
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
	formatKey := store.walletDB.Get(calcTxIndexKey(txID))
	if formatKey == nil {
		return nil, errAccntTxIDNotFound
	}
	rawTx := store.walletDB.Get(calcAnnotatedKey(string(formatKey)))
	tx := new(query.AnnotatedTx)
	if err := json.Unmarshal(rawTx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

// GetGlobalTransactionIndex get global tx by txID
func (store *WalletStore) GetGlobalTransactionIndex(txID string) []byte {
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
	if store.batch != nil {
		batch = store.batch
	}
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
	if store.batch == nil {
		batch.Write()
	}
}

// DeleteWalletUTXOs delete all txs in wallet
func (store *WalletStore) DeleteWalletUTXOs() {
	batch := store.walletDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}
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
	if store.batch == nil {
		batch.Write()
	}
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
