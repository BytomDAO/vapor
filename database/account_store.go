package database

import (
	"encoding/json"
	"strings"

	log "github.com/sirupsen/logrus"
	acc "github.com/vapor/account"
	"github.com/vapor/common"
	"github.com/vapor/crypto/ed25519/chainkd"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
)

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
func (store *AccountStore) InitBatch() error {
	if store.batch != nil {
		return errors.New("AccountStore initail fail, store batch is not nil.")
	}
	store.batch = store.accountDB.NewBatch()
	return nil
}

// CommitBatch commit batch
func (store *AccountStore) CommitBatch() error {
	if store.batch == nil {
		return errors.New("AccountStore commit fail, store batch is nil.")
	}
	store.batch.Write()
	store.batch = nil
	return nil
}

// DeleteAccountByAlias delete account by account alias
func (store *AccountStore) DeleteAccountByAlias(accountAlias string) {
	if store.batch == nil {
		store.accountDB.Delete(accountAliasKey(accountAlias))
	} else {
		store.batch.Delete(accountAliasKey(accountAlias))
	}
}

// DeleteAccount set account account ID, account alias and raw account.
func (store *AccountStore) DeleteAccount(account *acc.Account) {
	batch := store.accountDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}
	batch.Delete(AccountIDKey(account.ID))
	batch.Delete(accountAliasKey(account.Alias))
	if store.batch == nil {
		batch.Write()
	}
}

// DeleteControlProgram delete raw control program by hash
func (store *AccountStore) DeleteControlProgram(hash common.Hash) {
	if store.batch == nil {
		store.accountDB.Delete(ContractKey(hash))
	} else {
		store.batch.Delete(ContractKey(hash))
	}
}

// SetAccount set account account ID, account alias and raw account.
func (store *AccountStore) SetAccount(account *acc.Account) error {
	rawAccount, err := json.Marshal(account)
	if err != nil {
		return acc.ErrMarshalAccount
	}

	batch := store.accountDB.NewBatch()
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

// SetAccountIndex set account account ID, account alias and raw account.
func (store *AccountStore) SetAccountIndex(account *acc.Account) error {
	rawAccount, err := json.Marshal(account)
	if err != nil {
		return acc.ErrMarshalAccount
	}

	batch := store.accountDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}

	batch.Set(AccountIDKey(account.ID), rawAccount)
	batch.Set(accountAliasKey(account.Alias), []byte(account.ID))
	batch.Set(accountIndexKey(account.XPubs), common.Unit64ToBytes(account.KeyIndex))

	if store.batch == nil {
		batch.Write()
	}
	return nil
}

// GetAccountIDByAlias get account ID by account alias
func (store *AccountStore) GetAccountIDByAlias(accountAlias string) string {
	accountID := store.accountDB.Get(accountAliasKey(accountAlias))
	return string(accountID)
}

// GetAccountByID get account by accountID
func (store *AccountStore) GetAccountByID(accountID string) (*acc.Account, error) {
	rawAccount := store.accountDB.Get(AccountIDKey(accountID))
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
	if rawIndexBytes := store.accountDB.Get(accountIndexKey(xpubs)); rawIndexBytes != nil {
		currentIndex = common.BytesToUnit64(rawIndexBytes)
	}
	return currentIndex
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
		store.accountDB.Delete(contractIndexKey(accountID))
	} else {
		store.batch.Delete(contractIndexKey(accountID))
	}
}

// GetContractIndex get contract index
func (store *AccountStore) GetContractIndex(accountID string) uint64 {
	index := uint64(0)
	if rawIndexBytes := store.accountDB.Get(contractIndexKey(accountID)); rawIndexBytes != nil {
		index = common.BytesToUnit64(rawIndexBytes)
	}
	return index
}

// DeleteStandardUTXO delete utxo by outpu id
func (store *AccountStore) DeleteStandardUTXO(outputID bc.Hash) {
	if store.batch == nil {
		store.accountDB.Delete(StandardUTXOKey(outputID))
	} else {
		store.batch.Delete(StandardUTXOKey(outputID))
	}
}

// DeleteAccountUTXOs delete account utxos by accountID
func (store *AccountStore) DeleteAccountUTXOs(accountID string) error {
	batch := store.accountDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}

	accountUtxoIter := store.accountDB.IteratorPrefix([]byte(UTXOPrefix))
	defer accountUtxoIter.Release()

	for accountUtxoIter.Next() {
		accountUtxo := &acc.UTXO{}
		if err := json.Unmarshal(accountUtxoIter.Value(), accountUtxo); err != nil {
			return err
		}
		if accountID == accountUtxo.AccountID {
			batch.Delete(StandardUTXOKey(accountUtxo.OutputID))
		}
	}

	if store.batch == nil {
		batch.Write()
	}
	return nil
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
func (store *AccountStore) GetMiningAddress() (*acc.CtrlProgram, error) {
	rawCP := store.accountDB.Get([]byte(MiningAddressKey))
	if rawCP == nil {
		return nil, acc.ErrFindMiningAddress
	}
	cp := new(acc.CtrlProgram)
	if err := json.Unmarshal(rawCP, cp); err != nil {
		return nil, err
	}
	return cp, nil
}

// SetMiningAddress set mining address
func (store *AccountStore) SetMiningAddress(program *acc.CtrlProgram) error {
	rawProgram, err := json.Marshal(program)
	if err != nil {
		return err
	}

	if store.batch == nil {
		store.accountDB.Set([]byte(MiningAddressKey), rawProgram)
	} else {
		store.batch.Set([]byte(MiningAddressKey), rawProgram)
	}
	return nil
}

// GetBip44ContractIndex get bip44 contract index
func (store *AccountStore) GetBip44ContractIndex(accountID string, change bool) uint64 {
	index := uint64(0)
	if rawIndexBytes := store.accountDB.Get(Bip44ContractIndexKey(accountID, change)); rawIndexBytes != nil {
		index = common.BytesToUnit64(rawIndexBytes)
	}
	return index
}

// GetControlProgram get control program
func (store *AccountStore) GetControlProgram(hash common.Hash) (*acc.CtrlProgram, error) {
	rawProgram := store.accountDB.Get(ContractKey(hash))
	if rawProgram == nil {
		return nil, acc.ErrFindCtrlProgram
	}
	cp := new(acc.CtrlProgram)
	if err := json.Unmarshal(rawProgram, cp); err != nil {
		return nil, err
	}
	return cp, nil
}

// ListAccounts get all accounts which name prfix is id.
func (store *AccountStore) ListAccounts(id string) ([]*acc.Account, error) {
	accounts := []*acc.Account{}
	accountIter := store.accountDB.IteratorPrefix(AccountIDKey(strings.TrimSpace(id)))
	defer accountIter.Release()

	for accountIter.Next() {
		account := new(acc.Account)
		if err := json.Unmarshal(accountIter.Value(), &account); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

// ListControlPrograms get all local control programs
func (store *AccountStore) ListControlPrograms() ([]*acc.CtrlProgram, error) {
	cps := []*acc.CtrlProgram{}
	cpIter := store.accountDB.IteratorPrefix([]byte(ContractPrefix))
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

// SetControlProgram set raw program
func (store *AccountStore) SetControlProgram(hash common.Hash, program *acc.CtrlProgram) error {
	accountCP, err := json.Marshal(program)
	if err != nil {
		return err
	}
	if store.batch == nil {
		store.accountDB.Set(ContractKey(hash), accountCP)
	} else {
		store.batch.Set(ContractKey(hash), accountCP)
	}
	return nil
}

// SetContractIndex set contract index
func (store *AccountStore) SetContractIndex(accountID string, index uint64) {
	if store.batch == nil {
		store.accountDB.Set(contractIndexKey(accountID), common.Unit64ToBytes(index))
	} else {
		store.batch.Set(contractIndexKey(accountID), common.Unit64ToBytes(index))
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

// ListUTXOs get utxos by accountID
func (store *AccountStore) ListUTXOs() []*acc.UTXO {
	utxoIter := store.accountDB.IteratorPrefix([]byte(UTXOPrefix))
	defer utxoIter.Release()

	utxos := []*acc.UTXO{}
	for utxoIter.Next() {
		utxo := new(acc.UTXO)
		if err := json.Unmarshal(utxoIter.Value(), utxo); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("utxoKeeper findUtxos fail on unmarshal utxo")
			continue
		}
		utxos = append(utxos, utxo)
	}
	return utxos
}

// GetUTXO get standard utxo by id
func (store *AccountStore) GetUTXO(outid bc.Hash) (*acc.UTXO, error) {
	u := new(acc.UTXO)
	if data := store.accountDB.Get(StandardUTXOKey(outid)); data != nil {
		return u, json.Unmarshal(data, u)
	}
	if data := store.accountDB.Get(ContractUTXOKey(outid)); data != nil {
		return u, json.Unmarshal(data, u)
	}
	return nil, acc.ErrMatchUTXO
}

// SetStandardUTXO set standard utxo
func (store *AccountStore) SetStandardUTXO(outputID bc.Hash, utxo *acc.UTXO) error {
	data, err := json.Marshal(utxo)
	if err != nil {
		return err
	}
	if store.batch == nil {
		store.accountDB.Set(StandardUTXOKey(outputID), data)
	} else {
		store.batch.Set(StandardUTXOKey(outputID), data)
	}
	return nil
}
