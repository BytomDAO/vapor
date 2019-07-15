package orm

import (
	"github.com/vapor/toolbar/federation/types"
)

type Chain struct {
	ID          uint64 `gorm:"primary_key"`
	Name        string
	BlockHeight uint64
	BlockHash   string
	CreatedAt   types.Timestamp
	UpdatedAt   types.Timestamp
}
