package wallet

import (
	acc "github.com/vapor/account"
	"github.com/vapor/asset"
	"github.com/vapor/blockchain/query"
	"github.com/vapor/protocol/bc"
)

// WalletStore interface contains wallet storage functions.
type WalletStore interface {
	InitBatch() error
	CommitBatch() error
	DeleteContractUTXO(bc.Hash)
	DeleteRecoveryStatus()
	DeleteTransactions(uint64)
	DeleteUnconfirmedTransaction(string)
	DeleteWalletTransactions()
	DeleteWalletUTXOs()
	GetAsset(*bc.AssetID) (*asset.Asset, error)
	GetControlProgram(bc.Hash) (*acc.CtrlProgram, error) // delete it
	GetGlobalTransactionIndex(string) []byte
	GetStandardUTXO(bc.Hash) (*acc.UTXO, error)
	GetTransaction(string) (*query.AnnotatedTx, error)
	GetUnconfirmedTransaction(string) (*query.AnnotatedTx, error)
	GetRecoveryStatus([]byte) []byte     // recoveryManager.state isn't exported outside
	GetWalletInfo() (*StatusInfo, error) // need move database.NewWalletStore in wallet package
	ListAccountUTXOs(string) ([]*acc.UTXO, error)
	ListTransactions(string, string, uint, bool) ([]*query.AnnotatedTx, error)
	ListUnconfirmedTransactions() ([]*query.AnnotatedTx, error)
	SetAssetDefinition(*bc.AssetID, []byte)
	SetContractUTXO(bc.Hash, *acc.UTXO) error
	SetGlobalTransactionIndex(string, *bc.Hash, uint64)
	SetRecoveryStatus([]byte, []byte) // recoveryManager.state isn't exported outside
	SetTransaction(uint64, *query.AnnotatedTx) error
	SetUnconfirmedTransaction(string, *query.AnnotatedTx) error
	SetWalletInfo(*StatusInfo) error // need move database.NewWalletStore in wallet package
}
