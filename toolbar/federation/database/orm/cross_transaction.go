package orm

import (
	"database/sql"
	"encoding/json"

	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/toolbar/common"
	fedCommon "github.com/bytom/vapor/toolbar/federation/common"
)

type CrossTransaction struct {
	ID                   uint64 `gorm:"primary_key"`
	ChainID              uint64
	SourceBlockHeight    uint64
	SourceBlockTimestamp uint64
	SourceBlockHash      string
	SourceTxIndex        uint64
	SourceMuxID          string
	SourceTxHash         string
	SourceRawTransaction string
	DestBlockHeight      sql.NullInt64  `sql:"default:null"`
	DestBlockTimestamp   sql.NullInt64  `sql:"default:null"`
	DestBlockHash        sql.NullString `sql:"default:null"`
	DestTxIndex          sql.NullInt64  `sql:"default:null"`
	DestTxHash           sql.NullString `sql:"default:null"`
	Status               uint8
	CreatedAt            common.Timestamp
	UpdatedAt            common.Timestamp

	Chain *Chain `gorm:"foreignkey:ChainID"`
	Reqs  []*CrossTransactionReq
}

func (c *CrossTransaction) MarshalJSON() ([]byte, error) {
	var status string
	switch c.Status {
	case fedCommon.CrossTxPendingStatus:
		status = fedCommon.CrossTxPendingStatusLabel
	case fedCommon.CrossTxCompletedStatus:
		status = fedCommon.CrossTxCompletedStatusLabel
	default:
		return nil, errors.New("unknown cross-chain tx status")
	}

	return json.Marshal(&struct {
		SourceChainName      string                 `json:"source_chain_name"`
		SourceBlockHeight    uint64                 `json:"source_block_height"`
		SourceBlockTimestamp uint64                 `json:"source_block_timestamp"`
		SourceBlockHash      string                 `json:"source_block_hash"`
		SourceTxIndex        uint64                 `json:"source_tx_index"`
		SourceTxHash         string                 `json:"source_tx_hash"`
		DestBlockHeight      uint64                 `json:"dest_block_height"`
		DestBlockTimestamp   uint64                 `json:"dest_block_timestamp"`
		DestBlockHash        string                 `json:"dest_block_hash"`
		DestTxIndex          uint64                 `json:"dest_tx_index"`
		DestTxHash           string                 `json:"dest_tx_hash"`
		Status               string                 `json:"status"`
		Reqs                 []*CrossTransactionReq `json:"crosschain_requests"`
	}{
		SourceChainName:      c.Chain.Name,
		SourceBlockHeight:    c.SourceBlockHeight,
		SourceBlockTimestamp: c.SourceBlockTimestamp,
		SourceBlockHash:      c.SourceBlockHash,
		SourceTxIndex:        c.SourceTxIndex,
		SourceTxHash:         c.SourceTxHash,
		DestBlockHeight:      uint64(c.DestBlockHeight.Int64),
		DestBlockTimestamp:   uint64(c.DestBlockTimestamp.Int64),
		DestBlockHash:        c.DestBlockHash.String,
		DestTxIndex:          uint64(c.DestTxIndex.Int64),
		DestTxHash:           c.DestTxHash.String,
		Status:               status,
		Reqs:                 c.Reqs,
	})
}
