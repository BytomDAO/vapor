package orm

import (
	"github.com/vapor/federation/types"
)

type Asset struct {
	ID                uint64          `gorm:"primary_key" json:"-"`
	AssetID           string          `json:"-"`
	IssuanceProgram   string          `json:"-"`
	VMVersion         uint64          `json:"-"`
	RawDefinitionByte string          `json:"-"`
	CreatedAt         types.Timestamp `json:"-"`
	UpdatedAt         types.Timestamp `json:"-"`
}
