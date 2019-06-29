// Package account stores and tracks accounts within a Bytom Core.
package account

import (
	"reflect"
	"strings"
	"sync"

	"github.com/golang/groupcache/lru"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/blockchain/signers"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/common"
	"github.com/vapor/consensus"
	"github.com/vapor/consensus/segwit"
	"github.com/vapor/crypto"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/crypto/sha3pool"
	"github.com/vapor/errors"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/vm/vmutil"
)

const (
	maxAccountCache = 1000

	// HardenedKeyStart bip32 hierarchical deterministic wallets
	// keys with index ≥ 0x80000000 are hardened keys
	HardenedKeyStart = 0x80000000
	logModule        = "account"
)

// pre-define errors for supporting bytom errorFormatter
var (
	ErrDuplicateAlias    = errors.New("Duplicate account alias")
	ErrDuplicateIndex    = errors.New("Duplicate account with same xPubs and index")
	ErrFindAccount       = errors.New("Failed to find account")
	ErrMarshalAccount    = errors.New("Failed to marshal account")
	ErrInvalidAddress    = errors.New("Invalid address")
	ErrFindCtrlProgram   = errors.New("Failed to find account control program")
	ErrDeriveRule        = errors.New("Invalid key derivation rule")
	ErrContractIndex     = errors.New("Exceeded maximum addresses per account")
	ErrAccountIndex      = errors.New("Exceeded maximum accounts per xpub")
	ErrFindTransaction   = errors.New("No transaction")
	ErrFindMiningAddress = errors.New("Failed to find mining address")
)

// Account is structure of Bytom account
type Account struct {
	*signers.Signer
	ID    string `json:"id"`
	Alias string `json:"alias"`
}

//CtrlProgram is structure of account control program
type CtrlProgram struct {
	AccountID      string
	Address        string
	KeyIndex       uint64
	ControlProgram []byte
	Change         bool // Mark whether this control program is for UTXO change
}

// Manager stores accounts and their associated control programs.
type Manager struct {
	store      AccountStorer
	chain      *protocol.Chain
	utxoKeeper *utxoKeeper

	cacheMu    sync.Mutex
	cache      *lru.Cache
	aliasCache *lru.Cache

	delayedACPsMu sync.Mutex
	delayedACPs   map[*txbuilder.TemplateBuilder][]*CtrlProgram

	addressMu sync.Mutex
	accountMu sync.Mutex
}

// NewManager creates a new account manager
func NewManager(store AccountStorer, chain *protocol.Chain) *Manager {
	return &Manager{
		store:       store,
		chain:       chain,
		utxoKeeper:  newUtxoKeeper(chain.BestBlockHeight, store),
		cache:       lru.New(maxAccountCache),
		aliasCache:  lru.New(maxAccountCache),
		delayedACPs: make(map[*txbuilder.TemplateBuilder][]*CtrlProgram),
	}
}

// AddUnconfirmedUtxo add untxo list to utxoKeeper
func (m *Manager) AddUnconfirmedUtxo(utxos []*UTXO) {
	m.utxoKeeper.AddUnconfirmedUtxo(utxos)
}

// CreateAccount creates a new Account.
func CreateAccount(xpubs []chainkd.XPub, quorum int, alias string, acctIndex uint64, deriveRule uint8) (*Account, error) {
	if acctIndex >= HardenedKeyStart {
		return nil, ErrAccountIndex
	}

	signer, err := signers.Create("account", xpubs, quorum, acctIndex, deriveRule)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	id := signers.IDGenerate()
	return &Account{Signer: signer, ID: id, Alias: strings.ToLower(strings.TrimSpace(alias))}, nil
}

func (m *Manager) saveAccount(account *Account, updateIndex bool) error {
	m.store.InitBatch()

	if updateIndex {
		if err := m.store.SetAccountIndex(account); err != nil {
			return err
		}
	} else {
		if err := m.store.SetAccount(account); err != nil {
			return err
		}
	}

	m.store.CommitBatch()

	return nil
}

// SaveAccount save a new account.
func (m *Manager) SaveAccount(account *Account) error {
	m.accountMu.Lock()
	defer m.accountMu.Unlock()

	if existed := m.store.GetAccountIDByAlias(account.Alias); existed != "" {
		return ErrDuplicateAlias
	}

	acct, err := m.GetAccountByXPubsIndex(account.XPubs, account.KeyIndex)
	if err != nil {
		return err
	}

	if acct != nil {
		return ErrDuplicateIndex
	}

	currentIndex := m.store.GetAccountIndex(account.XPubs)
	return m.saveAccount(account, account.KeyIndex > currentIndex)
}

// Create creates and save a new Account.
func (m *Manager) Create(xpubs []chainkd.XPub, quorum int, alias string, deriveRule uint8) (*Account, error) {
	m.accountMu.Lock()
	defer m.accountMu.Unlock()

	if existed := m.store.GetAccountIDByAlias(alias); existed != "" {
		return nil, ErrDuplicateAlias
	}

	acctIndex := uint64(1)
	if currentIndex := m.store.GetAccountIndex(xpubs); currentIndex != 0 {
		acctIndex = currentIndex + 1
	}
	account, err := CreateAccount(xpubs, quorum, alias, acctIndex, deriveRule)
	if err != nil {
		return nil, err
	}

	if err := m.saveAccount(account, true); err != nil {
		return nil, err
	}

	return account, nil
}

func (m *Manager) UpdateAccountAlias(accountID string, newAlias string) error {
	m.accountMu.Lock()
	defer m.accountMu.Unlock()

	account, err := m.FindByID(accountID)
	if err != nil {
		return err
	}
	oldAlias := account.Alias

	normalizedAlias := strings.ToLower(strings.TrimSpace(newAlias))
	if existed := m.store.GetAccountIDByAlias(normalizedAlias); existed != "" {
		return ErrDuplicateAlias
	}

	m.cacheMu.Lock()
	m.aliasCache.Remove(oldAlias)
	m.cacheMu.Unlock()

	account.Alias = normalizedAlias

	m.store.InitBatch()

	m.store.DeleteAccountByAccountAlias(oldAlias)
	if err := m.store.SetAccount(account); err != nil {
		return err
	}

	m.store.CommitBatch()

	return nil
}

// CreateAddress generate an address for the select account
func (m *Manager) CreateAddress(accountID string, change bool) (cp *CtrlProgram, err error) {
	m.addressMu.Lock()
	defer m.addressMu.Unlock()

	account, err := m.FindByID(accountID)
	if err != nil {
		return nil, err
	}

	currentIdx, err := m.getCurrentContractIndex(account, change)
	if err != nil {
		return nil, err
	}

	cp, err = CreateCtrlProgram(account, currentIdx+1, change)
	if err != nil {
		return nil, err
	}

	return cp, m.saveControlProgram(cp, true)
}

// CreateBatchAddresses generate a batch of addresses for the select account
func (m *Manager) CreateBatchAddresses(accountID string, change bool, stopIndex uint64) error {
	m.addressMu.Lock()
	defer m.addressMu.Unlock()

	account, err := m.FindByID(accountID)
	if err != nil {
		return err
	}

	currentIndex, err := m.getCurrentContractIndex(account, change)
	if err != nil {
		return err
	}

	for currentIndex++; currentIndex <= stopIndex; currentIndex++ {
		cp, err := CreateCtrlProgram(account, currentIndex, change)
		if err != nil {
			return err
		}

		if err := m.saveControlProgram(cp, true); err != nil {
			return err
		}
	}

	return nil
}

// deleteAccountControlPrograms deletes control program matching accountID
func (m *Manager) deleteAccountControlPrograms(accountID string) error {
	cps, err := m.ListControlProgram()
	if err != nil {
		return err
	}

	var hash common.Hash
	for _, cp := range cps {
		if cp.AccountID == accountID {
			sha3pool.Sum256(hash[:], cp.ControlProgram)
			m.store.DeleteRawProgram(hash)
		}
	}

	m.store.DeleteBip44ContractIndex(accountID)
	m.store.DeleteContractIndex(accountID)

	return nil
}

// deleteAccountUtxos deletes utxos matching accountID
func (m *Manager) deleteAccountUtxos(accountID string) error {
	if err := m.store.DeleteAccountUTXOs(accountID); err != nil {
		return err
	}
	return nil
}

// DeleteAccount deletes the account's ID or alias matching account ID.
func (m *Manager) DeleteAccount(accountID string) (err error) {
	m.accountMu.Lock()
	defer m.accountMu.Unlock()

	account, err := m.FindByID(accountID)
	if err != nil {
		return err
	}

	if err := m.deleteAccountControlPrograms(accountID); err != nil {
		return err
	}
	if err := m.deleteAccountUtxos(accountID); err != nil {
		return err
	}

	m.cacheMu.Lock()
	m.aliasCache.Remove(account.Alias)
	m.cacheMu.Unlock()

	m.store.InitBatch()
	m.store.DeleteAccountByAccountAlias(account.Alias)
	m.store.DeleteAccountByAccountID(account.ID)
	m.store.CommitBatch()

	return nil
}

// FindByAlias retrieves an account's Signer record by its alias
func (m *Manager) FindByAlias(alias string) (*Account, error) {
	m.cacheMu.Lock()
	cachedID, ok := m.aliasCache.Get(alias)
	m.cacheMu.Unlock()
	if ok {
		return m.FindByID(cachedID.(string))
	}

	accountID := m.store.GetAccountIDByAlias(alias)
	if accountID == "" {
		return nil, ErrFindAccount
	}

	m.cacheMu.Lock()
	m.aliasCache.Add(alias, accountID)
	m.cacheMu.Unlock()
	return m.FindByID(accountID)
}

// FindByID returns an account's Signer record by its ID.
func (m *Manager) FindByID(id string) (*Account, error) {
	m.cacheMu.Lock()
	cachedAccount, ok := m.cache.Get(id)
	m.cacheMu.Unlock()
	if ok {
		return cachedAccount.(*Account), nil
	}

	account, err := m.store.GetAccountByAccountID(id)
	if err != nil {
		return nil, err
	}

	m.cacheMu.Lock()
	m.cache.Add(id, account)
	m.cacheMu.Unlock()
	return account, nil
}

// GetAccountByProgram return Account by given CtrlProgram
func (m *Manager) GetAccountByProgram(program *CtrlProgram) (*Account, error) {
	account, err := m.store.GetAccountByAccountID(program.AccountID)
	if err != nil {
		return nil, err
	}

	return account, nil
}

// GetAccountByXPubsIndex get account by xPubs and index
func (m *Manager) GetAccountByXPubsIndex(xPubs []chainkd.XPub, index uint64) (*Account, error) {
	accounts, err := m.ListAccounts("")
	if err != nil {
		return nil, err
	}

	for _, account := range accounts {
		if reflect.DeepEqual(account.XPubs, xPubs) && account.KeyIndex == index {
			return account, nil
		}
	}
	return nil, nil
}

// GetAliasByID return the account alias by given ID
func (m *Manager) GetAliasByID(id string) string {
	account, err := m.store.GetAccountByAccountID(id)
	if err != nil {
		log.Warn("GetAliasByID fail to find account")
		return ""
	}
	return account.Alias
}

func (m *Manager) GetCoinbaseArbitrary() []byte {
	if arbitrary := m.store.GetCoinbaseArbitrary(); arbitrary != nil {
		return arbitrary
	}
	return []byte{}
}

// GetCoinbaseControlProgram will return a coinbase script
func (m *Manager) GetCoinbaseControlProgram() ([]byte, error) {
	cp, err := m.GetCoinbaseCtrlProgram()
	if err == ErrFindAccount {
		log.Warningf("GetCoinbaseControlProgram: can't find any account in db")
		return vmutil.DefaultCoinbaseProgram()
	}
	if err != nil {
		return nil, err
	}
	return cp.ControlProgram, nil
}

// GetCoinbaseCtrlProgram will return the coinbase CtrlProgram
func (m *Manager) GetCoinbaseCtrlProgram() (*CtrlProgram, error) {
	cp, err := m.store.GetMiningAddress()
	if err != nil {
		return nil, err
	}
	if cp != nil {
		return cp, nil
	}

	account := new(Account)
	accounts, err := m.store.GetAccounts("")
	if err != nil {
		return nil, err
	}
	if len(accounts) > 0 {
		account = accounts[0]
	} else {
		return nil, ErrFindAccount
	}

	program, err := m.CreateAddress(account.ID, false)
	if err != nil {
		return nil, err
	}

	if err := m.store.SetMiningAddress(program); err != nil {
		return nil, err
	}

	return program, nil
}

// GetBip44ContractIndex return the current bip44 contract index
func (m *Manager) GetBip44ContractIndex(accountID string, change bool) uint64 {
	return m.store.GetBip44ContractIndex(accountID, change)
}

// GetLocalCtrlProgramByAddress return CtrlProgram by given address
func (m *Manager) GetLocalCtrlProgramByAddress(address string) (*CtrlProgram, error) {
	program, err := m.getProgramByAddress(address)
	if err != nil {
		return nil, err
	}

	var hash [32]byte
	sha3pool.Sum256(hash[:], program)

	cp, err := m.store.GetControlProgram(hash)
	if err != nil {
		return nil, err
	}
	if cp == nil {
		return nil, ErrFindCtrlProgram
	}

	return cp, nil
}

// GetMiningAddress will return the mining address
func (m *Manager) GetMiningAddress() (string, error) {
	cp, err := m.GetCoinbaseCtrlProgram()
	if err != nil {
		return "", err
	}
	return cp.Address, nil
}

// IsLocalControlProgram check is the input control program belong to local
func (m *Manager) IsLocalControlProgram(prog []byte) bool {
	var hash common.Hash
	sha3pool.Sum256(hash[:], prog)
	cp, err := m.store.GetControlProgram(hash)
	if err != nil || cp == nil {
		return false
	}
	return true
}

// ListAccounts will return the accounts in the db
func (m *Manager) ListAccounts(id string) ([]*Account, error) {
	return m.store.GetAccounts(id)
}

// ListControlProgram return all the local control program
func (m *Manager) ListControlProgram() ([]*CtrlProgram, error) {
	return m.store.GetControlPrograms()
}

func (m *Manager) ListUnconfirmedUtxo(accountID string, isSmartContract bool) []*UTXO {
	utxos := m.utxoKeeper.ListUnconfirmed()
	result := []*UTXO{}
	for _, utxo := range utxos {
		if segwit.IsP2WScript(utxo.ControlProgram) != isSmartContract && (accountID == utxo.AccountID || accountID == "") {
			result = append(result, utxo)
		}
	}
	return result
}

// RemoveUnconfirmedUtxo remove utxos from the utxoKeeper
func (m *Manager) RemoveUnconfirmedUtxo(hashes []*bc.Hash) {
	m.utxoKeeper.RemoveUnconfirmedUtxo(hashes)
}

// SetMiningAddress will set the mining address
func (m *Manager) SetMiningAddress(miningAddress string) (string, error) {
	program, err := m.getProgramByAddress(miningAddress)
	if err != nil {
		return "", err
	}

	cp := &CtrlProgram{
		Address:        miningAddress,
		ControlProgram: program,
	}

	if err := m.store.SetMiningAddress(cp); err != nil {
		return cp.Address, err
	}
	return m.GetMiningAddress()
}

func (m *Manager) SetCoinbaseArbitrary(arbitrary []byte) {
	m.store.SetCoinbaseArbitrary(arbitrary)
}

// CreateCtrlProgram generate an address for the select account
func CreateCtrlProgram(account *Account, addrIdx uint64, change bool) (cp *CtrlProgram, err error) {
	path, err := signers.Path(account.Signer, signers.AccountKeySpace, change, addrIdx)
	if err != nil {
		return nil, err
	}

	if len(account.XPubs) == 1 {
		cp, err = createP2PKH(account, path)
	} else {
		cp, err = createP2SH(account, path)
	}
	if err != nil {
		return nil, err
	}
	cp.KeyIndex, cp.Change = addrIdx, change
	return cp, nil
}

func createP2PKH(account *Account, path [][]byte) (*CtrlProgram, error) {
	derivedXPubs := chainkd.DeriveXPubs(account.XPubs, path)
	derivedPK := derivedXPubs[0].PublicKey()
	pubHash := crypto.Ripemd160(derivedPK)

	address, err := common.NewAddressWitnessPubKeyHash(pubHash, &consensus.ActiveNetParams)
	if err != nil {
		return nil, err
	}

	control, err := vmutil.P2WPKHProgram([]byte(pubHash))
	if err != nil {
		return nil, err
	}

	return &CtrlProgram{
		AccountID:      account.ID,
		Address:        address.EncodeAddress(),
		ControlProgram: control,
	}, nil
}

func createP2SH(account *Account, path [][]byte) (*CtrlProgram, error) {
	derivedXPubs := chainkd.DeriveXPubs(account.XPubs, path)
	derivedPKs := chainkd.XPubKeys(derivedXPubs)
	signScript, err := vmutil.P2SPMultiSigProgram(derivedPKs, account.Quorum)
	if err != nil {
		return nil, err
	}
	scriptHash := crypto.Sha256(signScript)

	address, err := common.NewAddressWitnessScriptHash(scriptHash, &consensus.ActiveNetParams)
	if err != nil {
		return nil, err
	}

	control, err := vmutil.P2WSHProgram(scriptHash)
	if err != nil {
		return nil, err
	}

	return &CtrlProgram{
		AccountID:      account.ID,
		Address:        address.EncodeAddress(),
		ControlProgram: control,
	}, nil
}

func (m *Manager) GetContractIndex(accountID string) uint64 {
	return m.store.GetContractIndex(accountID)
}

func (m *Manager) getCurrentContractIndex(account *Account, change bool) (uint64, error) {
	switch account.DeriveRule {
	case signers.BIP0032:
		return m.store.GetContractIndex(account.ID), nil
	case signers.BIP0044:
		return m.store.GetBip44ContractIndex(account.ID, change), nil
	}
	return 0, ErrDeriveRule
}

func (m *Manager) getProgramByAddress(address string) ([]byte, error) {
	addr, err := common.DecodeAddress(address, &consensus.ActiveNetParams)
	if err != nil {
		return nil, err
	}
	redeemContract := addr.ScriptAddress()
	program := []byte{}
	switch addr.(type) {
	case *common.AddressWitnessPubKeyHash:
		program, err = vmutil.P2WPKHProgram(redeemContract)
	case *common.AddressWitnessScriptHash:
		program, err = vmutil.P2WSHProgram(redeemContract)
	default:
		return nil, ErrInvalidAddress
	}
	if err != nil {
		return nil, err
	}
	return program, nil
}

func (m *Manager) saveControlProgram(prog *CtrlProgram, updateIndex bool) error {
	var hash common.Hash

	sha3pool.Sum256(hash[:], prog.ControlProgram)
	acct, err := m.GetAccountByProgram(prog)
	if err != nil {
		return err
	}

	m.store.InitBatch()
	if err := m.store.SetControlProgram(hash, prog); err != nil {
		return nil
	}
	if updateIndex {
		switch acct.DeriveRule {
		case signers.BIP0032:
			m.store.SetContractIndex(acct.ID, prog.KeyIndex)
		case signers.BIP0044:
			m.store.SetBip44ContractIndex(acct.ID, prog.Change, prog.KeyIndex)
		}
	}
	m.store.CommitBatch()
	return nil
}

// SaveControlPrograms save account control programs
func (m *Manager) SaveControlPrograms(progs ...*CtrlProgram) error {
	m.addressMu.Lock()
	defer m.addressMu.Unlock()

	for _, prog := range progs {
		acct, err := m.GetAccountByProgram(prog)
		if err != nil {
			return err
		}

		currentIndex, err := m.getCurrentContractIndex(acct, prog.Change)
		if err != nil {
			return err
		}

		m.saveControlProgram(prog, prog.KeyIndex > currentIndex)
	}
	return nil
}
