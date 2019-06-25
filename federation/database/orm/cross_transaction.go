package orm

import (
	"database/sql"

	"github.com/vapor/federation/types"
)

type CrossTransaction struct {
	ID                   uint64 `gorm:"primary_key"`
	ChainID              uint64
	SourceBlockHeight    uint64
	SourceBlockHash      string
	SourceTxIndex        uint64
	SourceMuxID          string
	SourceTxHash         string
	SourceRawTransaction string
	DestBlockHeight      sql.NullInt64
	DestBlockHash        sql.NullString
	DestTxIndex          sql.NullInt64
	DestTxHash           sql.NullString
	Status               uint8
	CreatedAt            types.Timestamp
	UpdatedAt            types.Timestamp

	Chain *Chain `gorm:"foreignkey:ChainID"`
	Reqs  []*CrossTransactionReq
}
