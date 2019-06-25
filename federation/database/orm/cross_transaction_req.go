package orm

import (
	"github.com/vapor/federation/types"
)

type CrossTransactionReq struct {
	ID                 uint64          `gorm:"primary_key" json:"-"`
	CrossTransactionID uint64          `json:"-"`
	SourcePos          uint64          `json:"-"`
	AssetID            uint64          `json:"-"`
	AssetAmount        uint64          `json:"-"`
	Script             string          `json:"-"`
	CreatedAt          types.Timestamp `json:"-"`
	UpdatedAt          types.Timestamp `json:"-"`

	CrossTransaction *CrossTransaction `gorm:"foreignkey:CrossTransactionID" json:"-"`
	Asset            *Asset            `gorm:"foreignkey:AssetID" json:"-"`
}
