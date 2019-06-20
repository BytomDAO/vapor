package wallet

import (
	"github.com/vapor/common"
	"github.com/vapor/protocol/bc"
)

// DB interface contains wallet storage functions.
type DB interface {
	GetAssetDefinitionByAssetID(*bc.AssetID) []byte
	GetRawProgramByAccountHash(common.Hash) []byte
	GetAccountValueByAccountID(string) []byte
}
