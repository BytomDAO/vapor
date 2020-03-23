package database

import (
	"encoding/json"
	"sort"
	"strings"

	acc "github.com/bytom/vapor/account"
	"github.com/bytom/vapor/blockchain/signers"
	"github.com/bytom/vapor/common"
	"github.com/bytom/vapor/crypto/ed25519/chainkd"
	"github.com/bytom/vapor/crypto/sha3pool"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
)

const (
	utxoPrefix byte = iota //UTXOPrefix is StandardUTXOKey prefix
	contractPrefix
	contractIndexPrefix
	accountPrefix // AccountPrefix is account ID prefix
	accountIndexPrefix
)

// leveldb key prefix
var (
	accountStore        = []byte("AS:")
	UTXOPrefix          = append(accountStore, utxoPrefix, colon)
	ContractPrefix      = append(accountStore, contractPrefix, colon)
	ContractIndexPrefix = append(accountStore, contractIndexPrefix, colon)
	AccountPrefix       = append(accountStore, accountPrefix, colon) // AccountPrefix is account ID prefix
	AccountIndexPrefix  = append(accountStore, accountIndexPrefix, colon)
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
	return append(AccountIndexPrefix, hash[:]...)
}

func bip44ContractIndexKey(accountID string, change bool) []byte {
	key := append(ContractIndexPrefix, []byte(accountID)...)
	if change {
		return append(key, 0x01)
	}
	return append(key, 0x00)
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

func accountAliasKey(name string) []byte {
	return append(AccountAliasPrefix, []byte(name)...)
}

// AccountStore satisfies AccountStore interface.
type AccountStore struct {
	db    dbm.DB
	batch dbm.Batch
}

// NewAccountStore create new AccountStore.
func NewAccountStore(db dbm.DB) *AccountStore {
	return &AccountStore{
		db:    db,
		batch: nil,
	}
}

// InitBatch initial new account store
func (store *AccountStore) InitBatch() acc.AccountStore {
	newStore := NewAccountStore(store.db)
	newStore.batch = newStore.db.NewBatch()
	return newStore
}

// CommitBatch commit batch
func (store *AccountStore) CommitBatch() error {
	if store.batch == nil {
		return errors.New("accountStore commit fail, store batch is nil")
	}
	store.batch.Write()
	store.batch = nil
	return nil
}

// DeleteAccount set account account ID, account alias and raw account.
func (store *AccountStore) DeleteAccount(account *acc.Account) error {
	batch := store.db.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}

	// delete account utxos
	store.deleteAccountUTXOs(account.ID, batch)

	// delete account control program
	if err := store.deleteAccountControlPrograms(account.ID, batch); err != nil {
		return err
	}

	// delete bip44 contract index
	batch.Delete(bip44ContractIndexKey(account.ID, false))
	batch.Delete(bip44ContractIndexKey(account.ID, true))

	// delete contract index
	batch.Delete(contractIndexKey(account.ID))

	// delete account id
	batch.Delete(AccountIDKey(account.ID))
	batch.Delete(accountAliasKey(account.Alias))
	if store.batch == nil {
		batch.Write()
	}
	return nil
}

// deleteAccountUTXOs delete account utxos by accountID
func (store *AccountStore) deleteAccountUTXOs(accountID string, batch dbm.Batch) error {
	accountUtxoIter := store.db.IteratorPrefix(UTXOPrefix)
	defer accountUtxoIter.Release()

	for accountUtxoIter.Next() {
		accountUtxo := new(acc.UTXO)
		if err := json.Unmarshal(accountUtxoIter.Value(), accountUtxo); err != nil {
			return err
		}

		if accountID == accountUtxo.AccountID {
			batch.Delete(StandardUTXOKey(accountUtxo.OutputID))
		}
	}

	return nil
}

// deleteAccountControlPrograms deletes account control program
func (store *AccountStore) deleteAccountControlPrograms(accountID string, batch dbm.Batch) error {
	cps, err := store.ListControlPrograms()
	if err != nil {
		return err
	}

	var hash [32]byte
	for _, cp := range cps {
		if cp.AccountID == accountID {
			sha3pool.Sum256(hash[:], cp.ControlProgram)
			batch.Delete(ContractKey(bc.NewHash(hash)))
		}
	}
	return nil
}

// DeleteStandardUTXO delete utxo by outpu id
func (store *AccountStore) DeleteStandardUTXO(outputID bc.Hash) {
	if store.batch == nil {
		store.db.Delete(StandardUTXOKey(outputID))
	} else {
		store.batch.Delete(StandardUTXOKey(outputID))
	}
}

// GetAccountByAlias get account by account alias
func (store *AccountStore) GetAccountByAlias(accountAlias string) (*acc.Account, error) {
	accountID := store.db.Get(accountAliasKey(accountAlias))
	if accountID == nil {
		return nil, acc.ErrFindAccount
	}

	return store.GetAccountByID(string(accountID))
}

// GetAccountByID get account by accountID
func (store *AccountStore) GetAccountByID(accountID string) (*acc.Account, error) {
	rawAccount := store.db.Get(AccountIDKey(accountID))
	if rawAccount == nil {
		return nil, acc.ErrFindAccount
	}

	account := new(acc.Account)
	if err := json.Unmarshal(rawAccount, account); err != nil {
		return nil, err
	}

	return account, nil
}

// GetAccountIndex get account index by account xpubs
func (store *AccountStore) GetAccountIndex(xpubs []chainkd.XPub) uint64 {
	currentIndex := uint64(0)
	if rawIndexBytes := store.db.Get(accountIndexKey(xpubs)); rawIndexBytes != nil {
		currentIndex = common.BytesToUnit64(rawIndexBytes)
	}
	return currentIndex
}

// GetBip44ContractIndex get bip44 contract index
func (store *AccountStore) GetBip44ContractIndex(accountID string, change bool) uint64 {
	index := uint64(0)
	if rawIndexBytes := store.db.Get(bip44ContractIndexKey(accountID, change)); rawIndexBytes != nil {
		index = common.BytesToUnit64(rawIndexBytes)
	}
	return index
}

// GetCoinbaseArbitrary get coinbase arbitrary
func (store *AccountStore) GetCoinbaseArbitrary() []byte {
	return store.db.Get(CoinbaseAbKey)
}

// GetContractIndex get contract index
func (store *AccountStore) GetContractIndex(accountID string) uint64 {
	index := uint64(0)
	if rawIndexBytes := store.db.Get(contractIndexKey(accountID)); rawIndexBytes != nil {
		index = common.BytesToUnit64(rawIndexBytes)
	}
	return index
}

// GetControlProgram get control program
func (store *AccountStore) GetControlProgram(hash bc.Hash) (*acc.CtrlProgram, error) {
	rawProgram := store.db.Get(ContractKey(hash))
	if rawProgram == nil {
		return nil, acc.ErrFindCtrlProgram
	}

	cp := new(acc.CtrlProgram)
	if err := json.Unmarshal(rawProgram, cp); err != nil {
		return nil, err
	}

	return cp, nil
}

// GetMiningAddress get mining address
func (store *AccountStore) GetMiningAddress() (*acc.CtrlProgram, error) {
	rawCP := store.db.Get(MiningAddressKey)
	if rawCP == nil {
		return nil, acc.ErrFindMiningAddress
	}

	cp := new(acc.CtrlProgram)
	if err := json.Unmarshal(rawCP, cp); err != nil {
		return nil, err
	}

	return cp, nil
}

// GetUTXO get standard utxo by id
func (store *AccountStore) GetUTXO(outid bc.Hash) (*acc.UTXO, error) {
	u := new(acc.UTXO)
	if data := store.db.Get(StandardUTXOKey(outid)); data != nil {
		return u, json.Unmarshal(data, u)
	}

	if data := store.db.Get(ContractUTXOKey(outid)); data != nil {
		return u, json.Unmarshal(data, u)
	}

	return nil, acc.ErrMatchUTXO
}

// ListAccounts get all accounts which name prfix is id.
func (store *AccountStore) ListAccounts(id string) ([]*acc.Account, error) {
	accounts := []*acc.Account{}
	accountIter := store.db.IteratorPrefix(AccountIDKey(strings.TrimSpace(id)))
	defer accountIter.Release()

	for accountIter.Next() {
		account := new(acc.Account)
		if err := json.Unmarshal(accountIter.Value(), account); err != nil {
			return nil, err
		}

		accounts = append(accounts, account)
	}
	return accounts, nil
}

// ListControlPrograms get all local control programs
func (store *AccountStore) ListControlPrograms() ([]*acc.CtrlProgram, error) {
	cps := []*acc.CtrlProgram{}
	cpIter := store.db.IteratorPrefix(ContractPrefix)
	defer cpIter.Release()

	for cpIter.Next() {
		cp := new(acc.CtrlProgram)
		if err := json.Unmarshal(cpIter.Value(), cp); err != nil {
			return nil, err
		}

		cps = append(cps, cp)
	}
	return cps, nil
}

// ListUTXOs list all utxos
func (store *AccountStore) ListUTXOs() ([]*acc.UTXO, error) {
	utxoIter := store.db.IteratorPrefix(UTXOPrefix)
	defer utxoIter.Release()

	utxos := []*acc.UTXO{}
	for utxoIter.Next() {
		utxo := new(acc.UTXO)
		if err := json.Unmarshal(utxoIter.Value(), utxo); err != nil {
			return nil, err
		}

		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

// SetAccount set account account ID, account alias and raw account.
func (store *AccountStore) SetAccount(account *acc.Account) error {
	rawAccount, err := json.Marshal(account)
	if err != nil {
		return acc.ErrMarshalAccount
	}

	batch := store.db.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}

	batch.Set(AccountIDKey(account.ID), rawAccount)
	batch.Set(accountAliasKey(account.Alias), []byte(account.ID))

	if store.batch == nil {
		batch.Write()
	}
	return nil
}

// SetAccountIndex update account index
func (store *AccountStore) SetAccountIndex(account *acc.Account) {
	currentIndex := store.GetAccountIndex(account.XPubs)
	if account.KeyIndex > currentIndex {
		if store.batch == nil {
			store.db.Set(accountIndexKey(account.XPubs), common.Unit64ToBytes(account.KeyIndex))
		} else {
			store.batch.Set(accountIndexKey(account.XPubs), common.Unit64ToBytes(account.KeyIndex))
		}
	}
}

// SetBip44ContractIndex set contract index
func (store *AccountStore) SetBip44ContractIndex(accountID string, change bool, index uint64) {
	if store.batch == nil {
		store.db.Set(bip44ContractIndexKey(accountID, change), common.Unit64ToBytes(index))
	} else {
		store.batch.Set(bip44ContractIndexKey(accountID, change), common.Unit64ToBytes(index))
	}
}

// SetCoinbaseArbitrary set coinbase arbitrary
func (store *AccountStore) SetCoinbaseArbitrary(arbitrary []byte) {
	if store.batch == nil {
		store.db.Set(CoinbaseAbKey, arbitrary)
	} else {
		store.batch.Set(CoinbaseAbKey, arbitrary)
	}
}

// SetContractIndex set contract index
func (store *AccountStore) SetContractIndex(accountID string, index uint64) {
	if store.batch == nil {
		store.db.Set(contractIndexKey(accountID), common.Unit64ToBytes(index))
	} else {
		store.batch.Set(contractIndexKey(accountID), common.Unit64ToBytes(index))
	}
}

// SetControlProgram set raw program
func (store *AccountStore) SetControlProgram(hash bc.Hash, program *acc.CtrlProgram) error {
	accountCP, err := json.Marshal(program)
	if err != nil {
		return err
	}
	if store.batch == nil {
		store.db.Set(ContractKey(hash), accountCP)
	} else {
		store.batch.Set(ContractKey(hash), accountCP)
	}
	return nil
}

// SetMiningAddress set mining address
func (store *AccountStore) SetMiningAddress(program *acc.CtrlProgram) error {
	rawProgram, err := json.Marshal(program)
	if err != nil {
		return err
	}

	if store.batch == nil {
		store.db.Set(MiningAddressKey, rawProgram)
	} else {
		store.batch.Set(MiningAddressKey, rawProgram)
	}
	return nil
}

// SetStandardUTXO set standard utxo
func (store *AccountStore) SetStandardUTXO(outputID bc.Hash, utxo *acc.UTXO) error {
	data, err := json.Marshal(utxo)
	if err != nil {
		return err
	}

	if store.batch == nil {
		store.db.Set(StandardUTXOKey(outputID), data)
	} else {
		store.batch.Set(StandardUTXOKey(outputID), data)
	}
	return nil
}
