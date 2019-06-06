package orm

import (
	"github.com/vapor/federation/types"
)

type Warder struct {
	ID        uint64 `gorm:"primary_key"`
	Pubkey    string
	CreatedAt types.Timestamp
	UpdatedAt types.Timestamp
}
