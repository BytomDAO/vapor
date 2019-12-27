package account

import (
	"github.com/bytom/vapor/crypto/ed25519/chainkd"
	"github.com/bytom/vapor/protocol/bc"
)

// AccountStore interface contains account storage functions.
type AccountStore interface {
	InitBatch() AccountStore
	CommitBatch() error
	DeleteAccount(*Account) error
	DeleteStandardUTXO(bc.Hash)
	GetAccountByAlias(string) (*Account, error)
	GetAccountByID(string) (*Account, error)
	GetAccountIndex([]chainkd.XPub) uint64
	GetBip44ContractIndex(string, bool) uint64
	GetCoinbaseArbitrary() []byte
	GetContractIndex(string) uint64
	GetControlProgram(bc.Hash) (*CtrlProgram, error)
	GetMiningAddress() (*CtrlProgram, error)
	GetUTXO(bc.Hash) (*UTXO, error)
	ListAccounts(string) ([]*Account, error)
	ListControlPrograms() ([]*CtrlProgram, error)
	ListUTXOs() ([]*UTXO, error)
	SetAccount(*Account) error
	SetAccountIndex(*Account)
	SetBip44ContractIndex(string, bool, uint64)
	SetCoinbaseArbitrary([]byte)
	SetContractIndex(string, uint64)
	SetControlProgram(bc.Hash, *CtrlProgram) error
	SetMiningAddress(*CtrlProgram) error
	SetStandardUTXO(bc.Hash, *UTXO) error
}
