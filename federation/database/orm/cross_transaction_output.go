package orm

import (
	"database/sql"

	"github.com/vapor/federation/types"
)

type CrossTransactionOutput struct {
	ID            uint64 `gorm:"primary_key"`
	SidechainTxID uint64
	MainchainTxID sql.NullInt64
	SourcePos     uint64
	AssetID       uint64
	AssetAmount   uint64
	Script        string
	CreatedAt     types.Timestamp
	UpdatedAt     types.Timestamp

	SidechainTransaction *CrossTransaction `gorm:"foreignkey:SidechainTxID"`
	MainchainTransaction *CrossTransaction `gorm:"foreignkey:MainchainTxID"`
	Asset                *Asset            `gorm:"foreignkey:AssetID"`
}
