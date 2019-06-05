package orm

import (
	"github.com/vapor/federation/types"
)

type CrossTransaction struct {
	ID             uint64 `gorm:"primary_key"`
	ChainID        uint64
	BlockHeight    uint64
	BlockHash      string
	TxIndex        uint64
	MuxID          string
	TxHash         string
	RawTransaction string
	Status         uint8
	CreatedAt      types.Timestamp
	UpdatedAt      types.Timestamp

	Chain *Chain `gorm:"foreignkey:ChainID"`
}
