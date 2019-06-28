package wallet

import (
	acc "github.com/vapor/account"
	"github.com/vapor/asset"
	"github.com/vapor/blockchain/query"
	"github.com/vapor/common"
	"github.com/vapor/protocol/bc"
)

// WalletStorer interface contains wallet storage functions.
type WalletStorer interface {
	InitBatch()
	CommitBatch()
	GetAssetDefinition(*bc.AssetID) (*asset.Asset, error)
	SetAssetDefinition(*bc.AssetID, []byte)
	GetControlProgram(common.Hash) (*acc.CtrlProgram, error)
	GetAccountByAccountID(string) (*acc.Account, error)
	DeleteTransactions(uint64)
	SetTransaction(uint64, *query.AnnotatedTx) error
	DeleteUnconfirmedTransaction(string)
	SetGlobalTransactionIndex(string, *bc.Hash, uint64)
	GetStandardUTXO(bc.Hash) (*acc.UTXO, error)
	GetTransaction(string) (*query.AnnotatedTx, error)
	GetGlobalTransactionIndex(string) []byte
	GetTransactions() ([]*query.AnnotatedTx, error)
	GetUnconfirmedTransactions() ([]*query.AnnotatedTx, error)
	GetUnconfirmedTransaction(string) (*query.AnnotatedTx, error)
	SetUnconfirmedTransaction(string, []byte)
	DeleteStardardUTXO(bc.Hash)
	DeleteContractUTXO(bc.Hash)
	SetStandardUTXO(bc.Hash, []byte)
	SetContractUTXO(bc.Hash, []byte)
	GetWalletInfo() []byte
	SetWalletInfo([]byte)
	DeleteWalletTransactions()
	DeleteWalletUTXOs()
	GetAccountUTXOs(key string) [][]byte
	SetRecoveryStatus([]byte, []byte)
	DeleteRecoveryStatus([]byte)
	GetRecoveryStatus([]byte) []byte
}
