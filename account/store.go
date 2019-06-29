package account

import (
	"github.com/vapor/common"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/protocol/bc"
)

// AccountStorer interface contains account storage functions.
type AccountStorer interface {
	InitBatch()
	CommitBatch()
	SetAccount(*Account) error
	SetAccountIndex(*Account) error
	GetAccountIDByAlias(string) string
	GetAccountByID(string) (*Account, error)
	GetAccountIndex([]chainkd.XPub) uint64
	DeleteAccountByAlias(string)
	DeleteAccount(*Account)
	DeleteControlProgram(common.Hash)
	DeleteBip44ContractIndex(string)
	DeleteContractIndex(string)
	GetContractIndex(string) uint64
	DeleteAccountUTXOs(string) error
	DeleteStandardUTXO(bc.Hash)
	GetCoinbaseArbitrary() []byte
	SetCoinbaseArbitrary([]byte)
	GetMiningAddress() (*CtrlProgram, error)
	SetMiningAddress(*CtrlProgram) error
	GetBip44ContractIndex(string, bool) uint64
	GetControlProgram(common.Hash) (*CtrlProgram, error)
	GetAccounts(string) ([]*Account, error)
	GetControlPrograms() ([]*CtrlProgram, error)
	SetControlProgram(common.Hash, *CtrlProgram) error
	SetContractIndex(string, uint64)
	SetBip44ContractIndex(string, bool, uint64)
	GetUTXOs() []*UTXO
	GetUTXO(bc.Hash) (*UTXO, error)
	SetStandardUTXO(bc.Hash, *UTXO) error
}
