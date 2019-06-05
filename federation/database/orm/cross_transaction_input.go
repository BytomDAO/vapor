package orm

import (
	"github.com/vapor/federation/types"
)

type CrossTransactionInput struct {
	ID        uint64 `gorm:"primary_key"`
	TxID      uint64
	SourcePos uint64
	CreatedAt types.Timestamp
	UpdatedAt types.Timestamp

	CrossTransaction *CrossTransaction `gorm:"foreignkey:TxID"`
}
