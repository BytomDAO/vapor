package orm

import (
	"github.com/vapor/toolbar/federation/types"
)

type CrossTransactionReq struct {
	ID                 uint64          `gorm:"primary_key" json:"-"`
	CrossTransactionID uint64          `json:"-"`
	SourcePos          uint64          `json:"-"`
	AssetID            uint64          `json:"-"`
	AssetAmount        uint64          `json:"amount"`
	Script             string          `json:"-"`
	FromAddress        string          `json:"from_address"`
	ToAddress          string          `json:"to_address"`
	CreatedAt          types.Timestamp `json:"-"`
	UpdatedAt          types.Timestamp `json:"-"`

	CrossTransaction *CrossTransaction `gorm:"foreignkey:CrossTransactionID" json:"-"`
	Asset            *Asset            `json:"asset"`
}
