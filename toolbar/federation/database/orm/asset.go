package orm

import (
	"github.com/vapor/toolbar/federation/types"
)

type Asset struct {
	ID              uint64          `gorm:"primary_key;foreignkey:ID" json:"-"`
	AssetID         string          `json:"asset_id"`
	IssuanceProgram string          `json:"-"`
	VMVersion       uint64          `json:"-"`
	Definition      string          `json:"-"`
	CreatedAt       types.Timestamp `json:"-"`
	UpdatedAt       types.Timestamp `json:"-"`
}
