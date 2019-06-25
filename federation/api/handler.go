package api

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/vapor/errors"
	"github.com/vapor/federation/common"
	"github.com/vapor/federation/database/orm"
)

func (s *Server) ListCrosschainTxs(c *gin.Context, listTxsReq *listCrosschainTxsReq, query *PaginationQuery) ([]*orm.CrossTransaction, error) {
	var ormTxs []*orm.CrossTransaction
	txFilter := &orm.CrossTransaction{}
	txQuery := s.db.Where(txFilter).Preload("Reqs")
	if listPending, err := listTxsReq.GetFilterBoolean("list_pending"); err == nil && listPending {
		txQuery = txQuery.Where("status = ?", common.CrossTxPendingStatus)
	}
	if listCompleted, err := listTxsReq.GetFilterBoolean("list_completed"); err == nil && listCompleted {
		txQuery = txQuery.Where("status = ?", common.CrossTxCompletedStatus)
	}
	if txHash, err := listTxsReq.GetFilterString("source_tx_hash"); err == nil && txHash != "" {
		txQuery = txQuery.Where("source_tx_hash = ?", txHash)
	}
	if txHash, err := listTxsReq.GetFilterString("dest_tx_hash"); err == nil && txHash != "" {
		txQuery = txQuery.Where("dest_tx_hash = ?", txHash)
	}
	txQuery = txQuery.Order(fmt.Sprintf("cross_transactions.source_block_height %s", listTxsReq.Sorter.Order))
	txQuery = txQuery.Order(fmt.Sprintf("cross_transactions.source_tx_index %s", listTxsReq.Sorter.Order))
	if err := txQuery.Offset(query.Start).Limit(query.Limit).Find(&ormTxs).Error; err != nil {
		return nil, errors.Wrap(err, "query txs")
	}

	return ormTxs, nil
}
