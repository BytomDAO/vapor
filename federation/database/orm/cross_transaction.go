package orm

import (
	"database/sql"

	"github.com/vapor/federation/types"
)

type CrossTransaction struct {
	ID                   uint64          `gorm:"primary_key" json:"-"`
	ChainID              uint64          `json:"-"`
	SourceBlockHeight    uint64          `json:"source_block_height"`
	SourceBlockHash      string          `json:"source_block_hash"`
	SourceTxIndex        uint64          `json:"source_tx_index"`
	SourceMuxID          string          `json:"-"`
	SourceTxHash         string          `json:"source_tx_hash"`
	SourceRawTransaction string          `json:"-"`
	DestBlockHeight      sql.NullInt64   `sql:"default:null" json:"dest_block_height"`
	DestBlockHash        sql.NullString  `sql:"default:null" json:"dest_block_hash"`
	DestTxIndex          sql.NullInt64   `sql:"default:null" json:"dest_tx_index"`
	DestTxHash           sql.NullString  `sql:"default:null" json:"dest_tx_hash"`
	Status               uint8           `json:"status"`
	CreatedAt            types.Timestamp `json:"-"`
	UpdatedAt            types.Timestamp `json:"-"`

	Chain *Chain                 `gorm:"foreignkey:ChainID" json:"-"`
	Reqs  []*CrossTransactionReq `json:"crosschain_requests"`
}
