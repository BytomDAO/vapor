package orm

import (
	"github.com/vapor/federation/types"
)

type Asset struct {
	ID                uint64 `gorm:"primary_key"`
	AssetID           string
	IssuanceProgram   string
	VMVersion         uint64
	RawDefinitionByte string
	CreatedAt         types.Timestamp
	UpdatedAt         types.Timestamp
}
