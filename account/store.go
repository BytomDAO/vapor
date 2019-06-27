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
	SetAccount(*Account, bool) error
	GetAccountIDByAccountAlias(string) string
	GetAccountByAccountID(string) (*Account, error)
	GetAccountIndex([]chainkd.XPub) uint64
	DeleteAccountByAccountAlias(string)
	DeleteAccountByAccountID(string)
	DeleteRawProgram(common.Hash)
	DeleteBip44ContractIndex(string)
	DeleteContractIndex(string)
	GetContractIndex(string) uint64
	DeleteAccountUTXOs(string) error
	DeleteStandardUTXO(bc.Hash)
	GetCoinbaseArbitrary() []byte
	SetCoinbaseArbitrary([]byte)
	GetMiningAddress() (*CtrlProgram, error)
	SetMiningAddress(*CtrlProgram) error
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