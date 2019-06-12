package orm

import (
	"github.com/vapor/federation/types"
)

type CrossTransactionReq struct {
	ID                 uint64 `gorm:"primary_key"`
	CrossTransactionID uint64
	SourcePos          uint64
	AssetID            uint64
	AssetAmount        uint64
	Script             string
	CreatedAt          types.Timestamp
	UpdatedAt          types.Timestamp

	CrossTransaction *CrossTransaction `gorm:"foreignkey:CrossTransactionID"`
	Asset            *Asset            `gorm:"foreignkey:AssetID"`
}
