package account

import (
	"encoding/json"
	"strings"

	log "github.com/sirupsen/logrus"
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
	DeleteAccountUTXOs(string) error
	GetCoinbaseArbitrary() []byte
	SetCoinbaseArbitrary([]byte)
	GetMiningAddress() []byte
	GetFirstAccount() (*Account, error)
	SetMiningAddress([]byte)
	GetBip44ContractIndex(string, bool) []byte
	GetRawProgram(common.Hash) []byte // duplicate in WalletStorer
	GetAccounts(string) ([]*Account, error)
	GetControlPrograms() ([]*CtrlProgram, error)
	SetRawProgram(common.Hash, []byte)
	SetContractIndex(string, uint64)
	SetBip44ContractIndex(string, bool, uint64)
	GetUTXOs(string) []*UTXO
	GetStandardUTXO(bc.Hash) []byte // duplicate in WalletStorer
	GetContractUTXO(bc.Hash) []byte
}

// AccountStore satisfies AccountStorer interface.
type AccountStore struct {
	DB dbm.DB
}

// NewAccountStore create new AccountStore.
func NewAccountStore(db dbm.DB) *AccountStore {
	return &AccountStore{
		DB: db,
	}
}

// SetAccount set account account ID, account alias and raw account.
func (store *AccountStore) SetAccount(accountID, accountAlias string, rawAccount []byte) {
	batch := store.DB.NewBatch()
	batch.Set(Key(accountID), rawAccount)
	batch.Set(aliasKey(accountAlias), []byte(accountID))
	batch.Write()
}

// SetAccountIndex set account index
func (store *AccountStore) SetAccountIndex(xpubs []chainkd.XPub, keyIndex uint64) {
	store.DB.Set(GetAccountIndexKey(xpubs), common.Unit64ToBytes(keyIndex))
}

// GetAccountByAccountAlias get account by account alias
func (store *AccountStore) GetAccountByAccountAlias(accountAlias string) []byte {
	return store.DB.Get(aliasKey(accountAlias))
}

// GetAccountByAccountID get account by accountID
func (store *AccountStore) GetAccountByAccountID(accountID string) []byte {
	return store.DB.Get(Key(accountID))
}

// GetAccountIndex get account index by account xpubs
func (store *AccountStore) GetAccountIndex(xpubs []chainkd.XPub) []byte {
	return store.DB.Get(GetAccountIndexKey(xpubs))
}

// DeleteAccountByAccountAlias delete account by account alias
func (store *AccountStore) DeleteAccountByAccountAlias(accountAlias string) {
	store.DB.Delete(aliasKey(accountAlias))
}

// DeleteAccountByAccountID delete account by accountID
func (store *AccountStore) DeleteAccountByAccountID(accountID string) {
	store.DB.Delete(Key(accountID))
}

// DeleteRawProgram delete raw control program by hash
func (store *AccountStore) DeleteRawProgram(hash common.Hash) {
	store.DB.Delete(ContractKey(hash))
}

// DeleteBip44ContractIndex delete bip44 contract index by accountID
func (store *AccountStore) DeleteBip44ContractIndex(accountID string) {
	batch := store.DB.NewBatch()
	batch.Delete(bip44ContractIndexKey(accountID, false))
	batch.Delete(bip44ContractIndexKey(accountID, true))
	batch.Write()
}

// DeleteContractIndex delete contract index by accountID
func (store *AccountStore) DeleteContractIndex(accountID string) {
	store.DB.Delete(contractIndexKey(accountID))
}

// GetContractIndex get contract index
func (store *AccountStore) GetContractIndex(accountID string) []byte {
	return store.DB.Get(contractIndexKey(accountID))
}

// DeleteAccountUTXOs delete account utxos by accountID
func (store *AccountStore) DeleteAccountUTXOs(accountID string) error {
	accountUtxoIter := store.DB.IteratorPrefix([]byte(UTXOPreFix))
	defer accountUtxoIter.Release()
	for accountUtxoIter.Next() {
		accountUtxo := &UTXO{}
		if err := json.Unmarshal(accountUtxoIter.Value(), accountUtxo); err != nil {
			return err
		}

		if accountID == accountUtxo.AccountID {
			store.DB.Delete(StandardUTXOKey(accountUtxo.OutputID))
		}
	}
	return nil
}

// GetCoinbaseArbitrary get coinbase arbitrary
func (store *AccountStore) GetCoinbaseArbitrary() []byte {
	return store.DB.Get(CoinbaseAbKey)
}

// SetCoinbaseArbitrary set coinbase arbitrary
func (store *AccountStore) SetCoinbaseArbitrary(arbitrary []byte) {
	store.DB.Set(CoinbaseAbKey, arbitrary)
}

// GetMiningAddress get mining address
func (store *AccountStore) GetMiningAddress() []byte {
	return store.DB.Get(miningAddressKey)
}

// GetFirstAccount get first account
func (store *AccountStore) GetFirstAccount() (*Account, error) {
	accountIter := store.DB.IteratorPrefix([]byte(accountPrefix))
	defer accountIter.Release()
	if !accountIter.Next() {
		return nil, ErrFindAccount
	}

	account := &Account{}
	if err := json.Unmarshal(accountIter.Value(), account); err != nil {
		return nil, err
	}
	return account, nil
}

// SetMiningAddress set mining address
func (store *AccountStore) SetMiningAddress(rawProgram []byte) {
	store.DB.Set(miningAddressKey, rawProgram)
}

// GetBip44ContractIndex get bip44 contract index
func (store *AccountStore) GetBip44ContractIndex(accountID string, change bool) []byte {
	return store.DB.Get(bip44ContractIndexKey(accountID, change))
}

// GetRawProgram get raw control program
func (store *AccountStore) GetRawProgram(hash common.Hash) []byte {
	return store.DB.Get(ContractKey(hash))
}

// GetAccounts get all accounts which name prfix is id.
func (store *AccountStore) GetAccounts(id string) ([]*Account, error) {
	accounts := []*Account{}
	accountIter := store.DB.IteratorPrefix(Key(strings.TrimSpace(id)))
	defer accountIter.Release()

	for accountIter.Next() {
		account := &Account{}
		if err := json.Unmarshal(accountIter.Value(), &account); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

// GetControlPrograms get all local control programs
func (store *AccountStore) GetControlPrograms() ([]*CtrlProgram, error) {
	cps := []*CtrlProgram{}
	cpIter := store.DB.IteratorPrefix(contractPrefix)
	defer cpIter.Release()

	for cpIter.Next() {
		cp := &CtrlProgram{}
		if err := json.Unmarshal(cpIter.Value(), cp); err != nil {
			return nil, err
		}
		cps = append(cps, cp)
	}
	return cps, nil
}

// SetRawProgram set raw program
func (store *AccountStore) SetRawProgram(hash common.Hash, program []byte) {
	store.DB.Set(ContractKey(hash), program)
}

// SetContractIndex set contract index
func (store *AccountStore) SetContractIndex(accountID string, index uint64) {
	store.DB.Set(contractIndexKey(accountID), common.Unit64ToBytes(index))
}

// SetBip44ContractIndex set contract index
func (store *AccountStore) SetBip44ContractIndex(accountID string, change bool, index uint64) {
	store.DB.Set(bip44ContractIndexKey(accountID, change), common.Unit64ToBytes(index))
}

// GetUTXOs get utxos by accountID
func (store *AccountStore) GetUTXOs(accountID string) []*UTXO {
	utxos := []*UTXO{}
	utxoIter := store.DB.IteratorPrefix([]byte(UTXOPreFix))
	defer utxoIter.Release()
	for utxoIter.Next() {
		u := &UTXO{}
		if err := json.Unmarshal(utxoIter.Value(), u); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("utxoKeeper findUtxos fail on unmarshal utxo")
			continue
		}
		utxos = append(utxos, u)
	}
	return utxos
}

// GetStandardUTXO get standard utxo by id
func (store *AccountStore) GetStandardUTXO(outid bc.Hash) []byte {
	return store.DB.Get(StandardUTXOKey(outid))
}

// GetContractUTXO get contract utxo
func (store *AccountStore) GetContractUTXO(outid bc.Hash) []byte {
	return store.DB.Get(ContractUTXOKey(outid))
}
