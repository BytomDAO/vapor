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
	SetAccount(string, string, []byte)
	SetAccountIndex([]chainkd.XPub, uint64)
	GetAccountByAccountAlias(string) []byte
	GetAccountByAccountID(string) []byte // duplicate in WalletStorer
	GetAccountIndex([]chainkd.XPub) []byte
	DeleteAccountByAccountAlias(string)
	DeleteAccountByAccountID(string)
	DeleteRawProgram(common.Hash)
	DeleteBip44ContractIndex(string)
	DeleteContractIndex(string)
	GetContractIndex(string) []byte
	// DeleteAccountUTXOs(string) error
	GetCoinbaseArbitrary() []byte
	SetCoinbaseArbitrary([]byte)
	GetMiningAddress() []byte
	GetFirstAccount() ([]byte, error)
	SetMiningAddress([]byte)
	GetBip44ContractIndex(string, bool) []byte
	GetRawProgram(common.Hash) []byte // duplicate in WalletStorer
	GetAccounts(string) ([][]byte, error)
	GetControlPrograms() ([][]byte, error)
	SetRawProgram(common.Hash, []byte)
	SetContractIndex(string, uint64)
	SetBip44ContractIndex(string, bool, uint64)
	GetUTXOs(string) [][]byte
	GetStandardUTXO(bc.Hash) []byte // duplicate in WalletStorer
	GetContractUTXO(bc.Hash) []byte
}

// AccountStore satisfies AccountStorer interface.
type AccountStore struct {
	accountDB dbm.DB
}

// NewAccountStore create new AccountStore.
func NewAccountStore(db dbm.DB) *AccountStore {
	return &AccountStore{
		accountDB: db,
	}
}

// SetAccount set account account ID, account alias and raw account.
func (store *AccountStore) SetAccount(accountID, accountAlias string, rawAccount []byte) {
	batch := store.accountDB.NewBatch()
	batch.Set(AccountIDKey(accountID), rawAccount)
	batch.Set(AccountAliasKey(accountAlias), []byte(accountID))
	batch.Write()
}

// SetAccountIndex set account index
func (store *AccountStore) SetAccountIndex(xpubs []chainkd.XPub, keyIndex uint64) {
	store.accountDB.Set(AccountIndexKey(xpubs), common.Unit64ToBytes(keyIndex))
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
	store.accountDB.Delete(AccountAliasKey(accountAlias))
}

// DeleteAccountByAccountID delete account by accountID
func (store *AccountStore) DeleteAccountByAccountID(accountID string) {
	store.accountDB.Delete(AccountIDKey(accountID))
}

// DeleteRawProgram delete raw control program by hash
func (store *AccountStore) DeleteRawProgram(hash common.Hash) {
	store.accountDB.Delete(ContractKey(hash))
}

// DeleteBip44ContractIndex delete bip44 contract index by accountID
func (store *AccountStore) DeleteBip44ContractIndex(accountID string) {
	batch := store.accountDB.NewBatch()
	batch.Delete(Bip44ContractIndexKey(accountID, false))
	batch.Delete(Bip44ContractIndexKey(accountID, true))
	batch.Write()
}

// DeleteContractIndex delete contract index by accountID
func (store *AccountStore) DeleteContractIndex(accountID string) {
	store.accountDB.Delete(ContractIndexKey(accountID))
}

// GetContractIndex get contract index
func (store *AccountStore) GetContractIndex(accountID string) []byte {
	return store.accountDB.Get(ContractIndexKey(accountID))
}

// // DeleteAccountUTXOs delete account utxos by accountID
// func (store *AccountStore) DeleteAccountUTXOs(accountID string) error {
// 	accountUtxoIter := store.accountDB.IteratorPrefix([]byte(UTXOPrefix))
// 	defer accountUtxoIter.Release()
// 	for accountUtxoIter.Next() {
// 		accountUtxo := &UTXO{}
// 		if err := json.Unmarshal(accountUtxoIter.Value(), accountUtxo); err != nil {
// 			return err
// 		}

// 		if accountID == accountUtxo.AccountID {
// 			store.accountDB.Delete(StandardUTXOKey(accountUtxo.OutputID))
// 		}
// 	}
// 	return nil
// }

// GetCoinbaseArbitrary get coinbase arbitrary
func (store *AccountStore) GetCoinbaseArbitrary() []byte {
	return store.accountDB.Get([]byte(CoinbaseAbKey))
}

// SetCoinbaseArbitrary set coinbase arbitrary
func (store *AccountStore) SetCoinbaseArbitrary(arbitrary []byte) {
	store.accountDB.Set([]byte(CoinbaseAbKey), arbitrary)
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
	store.accountDB.Set([]byte(MiningAddressKey), rawProgram)
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
func (store *AccountStore) GetAccounts(id string) ([][]byte, error) {
	accounts := make([][]byte, 0)
	accountIter := store.accountDB.IteratorPrefix(AccountIDKey(strings.TrimSpace(id)))
	defer accountIter.Release()

	for accountIter.Next() {
		accounts = append(accounts, accountIter.Value())
	}
	return accounts, nil
}

// GetControlPrograms get all local control programs
func (store *AccountStore) GetControlPrograms() ([][]byte, error) {
	cps := make([][]byte, 0)
	cpIter := store.accountDB.IteratorPrefix([]byte(ContractPrefix))
	defer cpIter.Release()

	for cpIter.Next() {
		cps = append(cps, cpIter.Value())
	}
	return cps, nil
}

// SetRawProgram set raw program
func (store *AccountStore) SetRawProgram(hash common.Hash, program []byte) {
	store.accountDB.Set(ContractKey(hash), program)
}

// SetContractIndex set contract index
func (store *AccountStore) SetContractIndex(accountID string, index uint64) {
	store.accountDB.Set(ContractIndexKey(accountID), common.Unit64ToBytes(index))
}

// SetBip44ContractIndex set contract index
func (store *AccountStore) SetBip44ContractIndex(accountID string, change bool, index uint64) {
	store.accountDB.Set(Bip44ContractIndexKey(accountID, change), common.Unit64ToBytes(index))
}

// GetUTXOs get utxos by accountID
func (store *AccountStore) GetUTXOs(accountID string) [][]byte {
	utxos := make([][]byte, 0)
	utxoIter := store.accountDB.IteratorPrefix([]byte(UTXOPrefix))
	defer utxoIter.Release()

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
