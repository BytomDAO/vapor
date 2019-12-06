package orm

import (
	"github.com/bytom/vapor/toolbar/common"
)

type Chain struct {
	ID          uint64           `gorm:"primary_key" json:"-"`
	Name        string           `json:"name"`
	BlockHeight uint64           `json:"block_height"`
	BlockHash   string           `json:"block_hash"`
	CreatedAt   common.Timestamp `json:"-"`
	UpdatedAt   common.Timestamp `json:"-"`
}
