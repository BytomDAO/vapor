package orm

import (
	"github.com/vapor/federation/types"
)

type CrossTransactionSign struct {
	ID                 uint64 `gorm:"primary_key"`
	CrossTransactionID uint64
	WarderID           uint8
	Signatures         string
	Status             uint8
	CreatedAt          types.Timestamp
	UpdatedAt          types.Timestamp

	CrossTransaction *CrossTransaction `gorm:"foreignkey:CrossTransactionID"`
	Warder           *Warder           `gorm:"foreignkey:WarderID"`
}
