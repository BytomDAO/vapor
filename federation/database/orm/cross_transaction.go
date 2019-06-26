package orm

import (
	"database/sql"
	"encoding/json"

	"github.com/vapor/errors"
	"github.com/vapor/federation/common"
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
	DestBlockHeight      sql.NullInt64  `sql:"default:null"`
	DestBlockHash        sql.NullString `sql:"default:null"`
	DestTxIndex          sql.NullInt64  `sql:"default:null"`
	DestTxHash           sql.NullString `sql:"default:null"`
	Status               uint8
	CreatedAt            types.Timestamp
	UpdatedAt            types.Timestamp

	Chain *Chain `gorm:"foreignkey:ChainID"`
	Reqs  []*CrossTransactionReq
}

func (c *CrossTransaction) MarshalJSON() ([]byte, error) {
	var status string
	switch c.Status {
	case common.CrossTxPendingStatus:
		status = common.CrossTxPendingStatusLabel
	case common.CrossTxCompletedStatus:
		status = common.CrossTxCompletedStatusLabel
	default:
		return nil, errors.New("unknown cross-chain tx status")
	}

	return json.Marshal(&struct {
		FromChain         string                 `json:"from_chain"`
		SourceBlockHeight uint64                 `json:"source_block_height"`
		SourceBlockHash   string                 `json:"source_block_hash"`
		SourceTxIndex     uint64                 `json:"source_tx_index"`
		SourceTxHash      string                 `json:"source_tx_hash"`
		DestBlockHeight   uint64                 `json:"dest_block_height"`
		DestBlockHash     string                 `json:"dest_block_hash"`
		DestTxIndex       uint64                 `json:"dest_tx_index"`
		DestTxHash        string                 `json:"dest_tx_hash"`
		Status            string                 `json:"status"`
		Reqs              []*CrossTransactionReq `json:"crosschain_requests"`
	}{
		FromChain:         c.Chain.Name,
		SourceBlockHeight: c.SourceBlockHeight,
		SourceBlockHash:   c.SourceBlockHash,
		SourceTxIndex:     c.SourceTxIndex,
		SourceTxHash:      c.SourceTxHash,
		DestBlockHeight:   uint64(c.DestBlockHeight.Int64),
		DestBlockHash:     c.DestBlockHash.String,
		DestTxIndex:       uint64(c.DestTxIndex.Int64),
		DestTxHash:        c.DestTxHash.String,
		Status:            status,
		Reqs:              c.Reqs,
	})
}
