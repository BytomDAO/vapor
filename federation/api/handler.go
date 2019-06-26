package api

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/vapor/errors"
	"github.com/vapor/federation/common"
	"github.com/vapor/federation/database/orm"
)

type listCrosschainTxsReq struct{ Display }

func (s *Server) ListCrosschainTxs(c *gin.Context, listTxsReq *listCrosschainTxsReq, query *PaginationQuery) ([]*orm.CrossTransaction, error) {
	var ormTxs []*orm.CrossTransaction
	txFilter := &orm.CrossTransaction{}

	// filter tx status
	if status, err := listTxsReq.GetFilterString("status"); err == nil && status != "" {
		switch strings.ToLower(status) {
		case common.CrossTxPendingStatusLabel:
			txFilter.Status = common.CrossTxPendingStatus
		case common.CrossTxCompletedStatusLabel:
			txFilter.Status = common.CrossTxCompletedStatus
		}
	}

	// filter tx hash
	if txHash, err := listTxsReq.GetFilterString("source_tx_hash"); err == nil && txHash != "" {
		txFilter.SourceTxHash = txHash
	}
	if txHash, err := listTxsReq.GetFilterString("dest_tx_hash"); err == nil && txHash != "" {
		txFilter.DestTxHash = sql.NullString{txHash, true}
	}

	txQuery := s.db.Preload("Chain").Preload("Reqs").Preload("Reqs.Asset").Where(txFilter)
	// filter direction
	if fromChainName, err := listTxsReq.GetFilterString("from_chain"); err == nil && fromChainName != "" {
		txQuery = txQuery.Joins("join chains on chains.id = cross_transactions.chain_id").Where("chains.name = ?", fromChainName)
	}
	txQuery = txQuery.Order(fmt.Sprintf("cross_transactions.source_block_height %s", listTxsReq.Sorter.Order))
	txQuery = txQuery.Order(fmt.Sprintf("cross_transactions.source_tx_index %s", listTxsReq.Sorter.Order))
	if err := txQuery.Offset(query.Start).Limit(query.Limit).Find(&ormTxs).Error; err != nil {
		return nil, errors.Wrap(err, "query txs")
	}

	return ormTxs, nil
}
