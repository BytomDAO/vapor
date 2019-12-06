package wallet

import (
	acc "github.com/bytom/vapor/account"
	"github.com/bytom/vapor/asset"
	"github.com/bytom/vapor/blockchain/query"
	"github.com/bytom/vapor/protocol/bc"
)

// WalletStore interface contains wallet storage functions.
type WalletStore interface {
	InitBatch() WalletStore
	CommitBatch() error
	DeleteContractUTXO(bc.Hash)
	DeleteRecoveryStatus()
	DeleteTransactions(uint64)
	DeleteUnconfirmedTransaction(string)
	DeleteWalletTransactions()
	DeleteWalletUTXOs()
	GetAsset(*bc.AssetID) (*asset.Asset, error)
	GetGlobalTransactionIndex(string) []byte
	GetStandardUTXO(bc.Hash) (*acc.UTXO, error)
	GetTransaction(string) (*query.AnnotatedTx, error)
	GetUnconfirmedTransaction(string) (*query.AnnotatedTx, error)
	GetRecoveryStatus() (*RecoveryState, error)
	GetWalletInfo() (*StatusInfo, error)
	ListAccountUTXOs(string, bool) ([]*acc.UTXO, error)
	ListTransactions(string, string, uint, bool) ([]*query.AnnotatedTx, error)
	ListUnconfirmedTransactions() ([]*query.AnnotatedTx, error)
	SetAssetDefinition(*bc.AssetID, []byte)
	SetContractUTXO(bc.Hash, *acc.UTXO) error
	SetGlobalTransactionIndex(string, *bc.Hash, uint64)
	SetRecoveryStatus(*RecoveryState) error
	SetTransaction(uint64, *query.AnnotatedTx) error
	SetUnconfirmedTransaction(string, *query.AnnotatedTx) error
	SetWalletInfo(*StatusInfo) error
}
