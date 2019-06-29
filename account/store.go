package account

import (
	"github.com/vapor/common"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/protocol/bc"
)

// AccountStorer interface contains account storage functions.
type AccountStorer interface {
	InitBatch() error
	CommitBatch() error
	DeleteAccount(*Account)
	DeleteAccountByAlias(string)
	DeleteAccountUTXOs(string) error
	DeleteBip44ContractIndex(string)
	DeleteContractIndex(string)
	DeleteControlProgram(common.Hash)
	DeleteStandardUTXO(bc.Hash)
	GetAccountByAlias(string) (*Account, error)
	GetAccountByID(string) (*Account, error)
	GetAccountIndex([]chainkd.XPub) uint64
	GetBip44ContractIndex(string, bool) uint64
	GetCoinbaseArbitrary() []byte
	GetContractIndex(string) uint64
	GetControlProgram(common.Hash) (*CtrlProgram, error)
	GetMiningAddress() (*CtrlProgram, error)
	GetUTXO(bc.Hash) (*UTXO, error)
	ListAccounts(string) ([]*Account, error)
	ListControlPrograms() ([]*CtrlProgram, error)
	ListUTXOs() []*UTXO
	SetAccount(*Account) error
	SetAccountIndex(*Account) error
	SetBip44ContractIndex(string, bool, uint64)
	SetCoinbaseArbitrary([]byte)
	SetContractIndex(string, uint64)
	SetControlProgram(common.Hash, *CtrlProgram) error
	SetMiningAddress(*CtrlProgram) error
	SetStandardUTXO(bc.Hash, *UTXO) error
}
