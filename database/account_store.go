package database

import (
	"strings"

	"github.com/vapor/common"
	"github.com/vapor/crypto/ed25519/chainkd"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/protocol/bc"
)

// AccountStorer interface contains account storage functions.
type AccountStorer interface {
	InitBatch()
	CommitBatch()
	SetAccount(string, string, []byte)
	SetAccountIndex([]chainkd.XPub, uint64)
	GetAccountByAccountAlias(string) []byte
	GetAccountByAccountID(string) []byte
	GetAccountIndex([]chainkd.XPub) []byte
	DeleteAccountByAccountAlias(string)
	DeleteAccountByAccountID(string)
	DeleteRawProgram(common.Hash)
	DeleteBip44ContractIndex(string)
	DeleteContractIndex(string)
	GetContractIndex(string) []byte
	GetAccountUTXOs(string) [][]byte
	DeleteStandardUTXO(bc.Hash)
	GetCoinbaseArbitrary() []byte
	SetCoinbaseArbitrary([]byte)
	GetMiningAddress() []byte
	GetFirstAccount() ([]byte, error)
	SetMiningAddress([]byte)
	GetBip44ContractIndex(string, bool) []byte
	GetRawProgram(common.Hash) []byte
	GetAccounts(string) [][]byte
	GetControlPrograms() ([][]byte, error)
	SetRawProgram(common.Hash, []byte)
	SetContractIndex(string, uint64)
	SetBip44ContractIndex(string, bool, uint64)
	GetUTXOs() [][]byte
	GetStandardUTXO(bc.Hash) []byte
	GetContractUTXO(bc.Hash) []byte
	SetStandardUTXO(bc.Hash, []byte)
}

// AccountStore satisfies AccountStorer interface.
type AccountStore struct {
	accountDB dbm.DB
	batch     dbm.Batch
}

// NewAccountStore create new AccountStore.
func NewAccountStore(db dbm.DB) *AccountStore {
	return &AccountStore{
		accountDB: db,
		batch:     nil,
	}
}

// InitBatch initial batch
func (store *AccountStore) InitBatch() {
	if store.batch == nil {
		store.batch = store.accountDB.NewBatch()
	}
}

// CommitBatch commit batch
func (store *AccountStore) CommitBatch() {
	if store.batch != nil {
		store.batch.Write()
		store.batch = nil
	}
}

// SetAccount set account account ID, account alias and raw account.
func (store *AccountStore) SetAccount(accountID, accountAlias string, rawAccount []byte) {
	batch := store.accountDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}
	batch.Set(AccountIDKey(accountID), rawAccount)
	batch.Set(AccountAliasKey(accountAlias), []byte(accountID))
	if store.batch == nil {
		batch.Write()
	}
}

// DeleteAccount set account account ID, account alias and raw account.
func (store *AccountStore) DeleteAccount(accountID, accountAlias string) {
	batch := store.accountDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}
	batch.Delete(AccountIDKey(accountID))
	batch.Delete(AccountAliasKey(accountAlias))
	if store.batch == nil {
		batch.Write()
	}
}

// SetAccountIndex set account index
func (store *AccountStore) SetAccountIndex(xpubs []chainkd.XPub, keyIndex uint64) {
	if store.batch == nil {
		store.accountDB.Set(AccountIndexKey(xpubs), common.Unit64ToBytes(keyIndex))
	} else {
		store.batch.Set(AccountIndexKey(xpubs), common.Unit64ToBytes(keyIndex))
	}
}

// GetAccountByAccountAlias get account by account alias
func (store *AccountStore) GetAccountByAccountAlias(accountAlias string) []byte {
	return store.accountDB.Get(AccountAliasKey(accountAlias))
}

// GetAccountByAccountID get account by accountID
func (store *AccountStore) GetAccountByAccountID(accountID string) []byte {
	return store.accountDB.Get(AccountIDKey(accountID))
}

// GetAccountIndex get account index by account xpubs
func (store *AccountStore) GetAccountIndex(xpubs []chainkd.XPub) []byte {
	return store.accountDB.Get(AccountIndexKey(xpubs))
}

// DeleteAccountByAccountAlias delete account by account alias
func (store *AccountStore) DeleteAccountByAccountAlias(accountAlias string) {
	if store.batch == nil {
		store.accountDB.Delete(AccountAliasKey(accountAlias))
	} else {
		store.batch.Delete(AccountAliasKey(accountAlias))
	}
}

// DeleteAccountByAccountID delete account by accountID
func (store *AccountStore) DeleteAccountByAccountID(accountID string) {
	if store.batch == nil {
		store.accountDB.Delete(AccountIDKey(accountID))
	} else {
		store.batch.Delete(AccountIDKey(accountID))
	}
}

// DeleteRawProgram delete raw control program by hash
func (store *AccountStore) DeleteRawProgram(hash common.Hash) {
	if store.batch == nil {
		store.accountDB.Delete(ContractKey(hash))
	} else {
		store.batch.Delete(ContractKey(hash))
	}
}

// DeleteBip44ContractIndex delete bip44 contract index by accountID
func (store *AccountStore) DeleteBip44ContractIndex(accountID string) {
	batch := store.accountDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}
	batch.Delete(Bip44ContractIndexKey(accountID, false))
	batch.Delete(Bip44ContractIndexKey(accountID, true))
	if store.batch == nil {
		batch.Write()
	}
}

// DeleteContractIndex delete contract index by accountID
func (store *AccountStore) DeleteContractIndex(accountID string) {
	if store.batch == nil {
		store.accountDB.Delete(ContractIndexKey(accountID))
	} else {
		store.batch.Delete(ContractIndexKey(accountID))
	}
}

// GetContractIndex get contract index
func (store *AccountStore) GetContractIndex(accountID string) []byte {
	return store.accountDB.Get(ContractIndexKey(accountID))
}

// GetAccountUTXOs get account utxos by account id
func (store *AccountStore) GetAccountUTXOs(accountID string) [][]byte {
	accountUtxoIter := store.accountDB.IteratorPrefix([]byte(UTXOPrefix))
	defer accountUtxoIter.Release()

	utxos := make([][]byte, 0)
	for accountUtxoIter.Next() {
		utxos = append(utxos, accountUtxoIter.Value())
	}
	return utxos
}

// DeleteStandardUTXO delete utxo by outpu id
func (store *AccountStore) DeleteStandardUTXO(outputID bc.Hash) {
	if store.batch == nil {
		store.accountDB.Delete(StandardUTXOKey(outputID))
	} else {
		store.batch.Delete(StandardUTXOKey(outputID))
	}
}

// GetCoinbaseArbitrary get coinbase arbitrary
func (store *AccountStore) GetCoinbaseArbitrary() []byte {
	return store.accountDB.Get([]byte(CoinbaseAbKey))
}

// SetCoinbaseArbitrary set coinbase arbitrary
func (store *AccountStore) SetCoinbaseArbitrary(arbitrary []byte) {
	if store.batch == nil {
		store.accountDB.Set([]byte(CoinbaseAbKey), arbitrary)
	} else {
		store.batch.Set([]byte(CoinbaseAbKey), arbitrary)
	}
}

// GetMiningAddress get mining address
func (store *AccountStore) GetMiningAddress() []byte {
	return store.accountDB.Get([]byte(MiningAddressKey))
}

// GetFirstAccount get first account
func (store *AccountStore) GetFirstAccount() ([]byte, error) {
	accountIter := store.accountDB.IteratorPrefix([]byte(AccountPrefix))
	defer accountIter.Release()

	if !accountIter.Next() {
		return nil, ErrFindAccount
	}
	return accountIter.Value(), nil
}

// SetMiningAddress set mining address
func (store *AccountStore) SetMiningAddress(rawProgram []byte) {
	if store.batch == nil {
		store.accountDB.Set([]byte(MiningAddressKey), rawProgram)
	} else {
		store.batch.Set([]byte(MiningAddressKey), rawProgram)
	}
}

// GetBip44ContractIndex get bip44 contract index
func (store *AccountStore) GetBip44ContractIndex(accountID string, change bool) []byte {
	return store.accountDB.Get(Bip44ContractIndexKey(accountID, change))
}

// GetRawProgram get raw control program
func (store *AccountStore) GetRawProgram(hash common.Hash) []byte {
	return store.accountDB.Get(ContractKey(hash))
}

// GetAccounts get all accounts which name prfix is id.
func (store *AccountStore) GetAccounts(id string) [][]byte {
	accountIter := store.accountDB.IteratorPrefix(AccountIDKey(strings.TrimSpace(id)))
	defer accountIter.Release()

	accounts := make([][]byte, 0)
	for accountIter.Next() {
		accounts = append(accounts, accountIter.Value())
	}
	return accounts
}

// GetControlPrograms get all local control programs
func (store *AccountStore) GetControlPrograms() ([][]byte, error) {
	cpIter := store.accountDB.IteratorPrefix([]byte(ContractPrefix))
	defer cpIter.Release()

	cps := make([][]byte, 0)
	for cpIter.Next() {
		cps = append(cps, cpIter.Value())
	}
	return cps, nil
}

// SetRawProgram set raw program
func (store *AccountStore) SetRawProgram(hash common.Hash, program []byte) {
	if store.batch == nil {
		store.accountDB.Set(ContractKey(hash), program)
	} else {
		store.batch.Set(ContractKey(hash), program)
	}
}

// SetContractIndex set contract index
func (store *AccountStore) SetContractIndex(accountID string, index uint64) {
	if store.batch == nil {
		store.accountDB.Set(ContractIndexKey(accountID), common.Unit64ToBytes(index))
	} else {
		store.batch.Set(ContractIndexKey(accountID), common.Unit64ToBytes(index))
	}
}

// SetBip44ContractIndex set contract index
func (store *AccountStore) SetBip44ContractIndex(accountID string, change bool, index uint64) {
	if store.batch == nil {
		store.accountDB.Set(Bip44ContractIndexKey(accountID, change), common.Unit64ToBytes(index))
	} else {
		store.batch.Set(Bip44ContractIndexKey(accountID, change), common.Unit64ToBytes(index))
	}
}

// GetUTXOs get utxos by accountID
func (store *AccountStore) GetUTXOs() [][]byte {
	utxoIter := store.accountDB.IteratorPrefix([]byte(UTXOPrefix))
	defer utxoIter.Release()

	utxos := make([][]byte, 0)
	for utxoIter.Next() {
		utxos = append(utxos, utxoIter.Value())
	}
	return utxos
}

// GetStandardUTXO get standard utxo by id
func (store *AccountStore) GetStandardUTXO(outid bc.Hash) []byte {
	return store.accountDB.Get(StandardUTXOKey(outid))
}

// GetContractUTXO get contract utxo
func (store *AccountStore) GetContractUTXO(outid bc.Hash) []byte {
	return store.accountDB.Get(ContractUTXOKey(outid))
}

// SetStandardUTXO set standard utxo
func (store *AccountStore) SetStandardUTXO(outputID bc.Hash, data []byte) {
	if store.batch == nil {
		store.accountDB.Set(StandardUTXOKey(outputID), data)
	} else {
		store.batch.Set(StandardUTXOKey(outputID), data)
	}
}
