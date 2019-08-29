package orm

import (
	"github.com/vapor/toolbar/federation/types"
)

type Chain struct {
	ID          uint64          `gorm:"primary_key" json:"-"`
	Name        string          `json:"name"`
	BlockHeight uint64          `json:"block_height"`
	BlockHash   string          `json:"block_hash"`
	CreatedAt   types.Timestamp `json:"-"`
	UpdatedAt   types.Timestamp `json:"-"`
}
