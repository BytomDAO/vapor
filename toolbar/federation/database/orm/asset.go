package orm

import (
	"github.com/vapor/toolbar/common"
)

type Asset struct {
	ID              uint64           `gorm:"primary_key;foreignkey:ID" json:"-"`
	AssetID         string           `json:"asset_id"`
	IssuanceProgram string           `json:"-"`
	VMVersion       uint64           `json:"-"`
	Definition      string           `json:"-"`
	IsFilter        bool             `json:"_"`
	CreatedAt       common.Timestamp `json:"-"`
	UpdatedAt       common.Timestamp `json:"-"`
}

type FilterAsset struct {
	ID      uint64 `gorm:"primary_key" json:"-"`
	AssetID string `json:"asset_id"`
}
