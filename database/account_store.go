package database

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	acc "github.com/vapor/account"
	"github.com/vapor/common"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/crypto/sha3pool"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
)

// AccountStore satisfies AccountStore interface.
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

// DeleteAccount set account account ID, account alias and raw account.
func (store *AccountStore) DeleteAccount(account *acc.Account) error {
	batch := store.accountDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}

	// delete account utxos
	store.deleteAccountUTXOs(account.ID)

	// delete account control program
	cps, err := store.ListControlPrograms()
	if err != nil {
		return err
	}
	var hash [32]byte
	for _, cp := range cps {
		if cp.AccountID == account.ID {
			sha3pool.Sum256(hash[:], cp.ControlProgram)
			batch.Delete(ContractKey(bc.NewHash(hash)))
		}
	}

	// delete bip44 contract index
	batch.Delete(Bip44ContractIndexKey(account.ID, false))
	batch.Delete(Bip44ContractIndexKey(account.ID, true))

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
func (store *AccountStore) deleteAccountUTXOs(accountID string) error {
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

// DeleteStandardUTXO delete utxo by outpu id
func (store *AccountStore) DeleteStandardUTXO(outputID bc.Hash) {
	if store.batch == nil {
		store.accountDB.Delete(StandardUTXOKey(outputID))
	} else {
		store.batch.Delete(StandardUTXOKey(outputID))
	}
}

// GetAccountByAlias get account by account alias
func (store *AccountStore) GetAccountByAlias(accountAlias string) (*acc.Account, error) {
	accountID := store.accountDB.Get(accountAliasKey(accountAlias))
	if accountID == nil {
		return nil, acc.ErrFindAccount
	}
	return store.GetAccountByID(string(accountID))
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

// GetBip44ContractIndex get bip44 contract index
func (store *AccountStore) GetBip44ContractIndex(accountID string, change bool) uint64 {
	index := uint64(0)
	if rawIndexBytes := store.accountDB.Get(Bip44ContractIndexKey(accountID, change)); rawIndexBytes != nil {
		index = common.BytesToUnit64(rawIndexBytes)
	}
	return index
}

// GetCoinbaseArbitrary get coinbase arbitrary
func (store *AccountStore) GetCoinbaseArbitrary() []byte {
	return store.accountDB.Get([]byte(CoinbaseAbKey))
}

// GetContractIndex get contract index
func (store *AccountStore) GetContractIndex(accountID string) uint64 {
	index := uint64(0)
	if rawIndexBytes := store.accountDB.Get(contractIndexKey(accountID)); rawIndexBytes != nil {
		index = common.BytesToUnit64(rawIndexBytes)
	}
	return index
}

// GetControlProgram get control program
func (store *AccountStore) GetControlProgram(hash bc.Hash) (*acc.CtrlProgram, error) {
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
	// cpIter := store.accountDB.IteratorPrefix([]byte{0x02, 0x3a})
	defer cpIter.Release()

	for cpIter.Next() {
		cp := new(acc.CtrlProgram)
		v := hex.EncodeToString(cpIter.Value())
		fmt.Println("v:", v)
		if err := json.Unmarshal(cpIter.Value(), cp); err != nil {
			return nil, err
		}
		cps = append(cps, cp)
	}
	return cps, nil
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
func (store *AccountStore) SetAccountIndex(account *acc.Account) {
	if store.batch == nil {
		store.accountDB.Set(accountIndexKey(account.XPubs), common.Unit64ToBytes(account.KeyIndex))
	} else {
		store.batch.Set(accountIndexKey(account.XPubs), common.Unit64ToBytes(account.KeyIndex))
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

// SetCoinbaseArbitrary set coinbase arbitrary
func (store *AccountStore) SetCoinbaseArbitrary(arbitrary []byte) {
	if store.batch == nil {
		store.accountDB.Set([]byte(CoinbaseAbKey), arbitrary)
	} else {
		store.batch.Set([]byte(CoinbaseAbKey), arbitrary)
	}
}

// SetContractIndex set contract index
func (store *AccountStore) SetContractIndex(accountID string, index uint64) {
	if store.batch == nil {
		store.accountDB.Set(contractIndexKey(accountID), common.Unit64ToBytes(index))
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
		store.accountDB.Set(ContractKey(hash), accountCP)
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
		store.accountDB.Set([]byte(MiningAddressKey), rawProgram)
	} else {
		store.batch.Set([]byte(MiningAddressKey), rawProgram)
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
		store.accountDB.Set(StandardUTXOKey(outputID), data)
	} else {
		store.batch.Set(StandardUTXOKey(outputID), data)
	}
	return nil
}
