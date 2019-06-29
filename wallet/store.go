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
	InitBatch() error
	CommitBatch() error
	DeleteContractUTXO(bc.Hash)
	DeleteRecoveryStatus()
	DeleteStardardUTXO(bc.Hash)
	DeleteTransactions(uint64)
	DeleteUnconfirmedTransaction(string)
	DeleteWalletTransactions()
	DeleteWalletUTXOs()
	GetAccountByID(string) (*acc.Account, error)
	GetAssetDefinition(*bc.AssetID) (*asset.Asset, error)
	GetControlProgram(common.Hash) (*acc.CtrlProgram, error)
	GetGlobalTransactionIndex(string) []byte
	GetStandardUTXO(bc.Hash) (*acc.UTXO, error)
	GetTransaction(string) (*query.AnnotatedTx, error)
	GetUnconfirmedTransaction(string) (*query.AnnotatedTx, error)
	ListUnconfirmedTransactions() ([]*query.AnnotatedTx, error)
	GetRecoveryStatus([]byte) []byte // recoveryManager.state isn't exported outside
	GetWalletInfo() []byte           // need move database.NewWalletStore in wallet package
	ListAccountUTXOs(string) ([]*acc.UTXO, error)
	ListTransactions() ([]*query.AnnotatedTx, error)
	SetAssetDefinition(*bc.AssetID, []byte)
	SetTransaction(uint64, *query.AnnotatedTx) error
	SetGlobalTransactionIndex(string, *bc.Hash, uint64)
	SetUnconfirmedTransaction(string, *query.AnnotatedTx) error
	SetStandardUTXO(bc.Hash, *acc.UTXO) error
	SetContractUTXO(bc.Hash, *acc.UTXO) error
	SetWalletInfo([]byte)             // need move database.NewWalletStore in wallet package
	SetRecoveryStatus([]byte, []byte) // recoveryManager.state isn't exported outside
}
