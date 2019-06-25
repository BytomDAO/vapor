package api

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/vapor/errors"
	"github.com/vapor/federation/common"
	"github.com/vapor/federation/database/orm"
)

type listCrosschainTxsReq struct{ Display }

func (s *Server) ListCrosschainTxs(c *gin.Context, listTxsReq *listCrosschainTxsReq, query *PaginationQuery) ([]*orm.CrossTransaction, error) {
	var ormTxs []*orm.CrossTransaction
	txFilter := &orm.CrossTransaction{}
	if listPending, err := listTxsReq.GetFilterBoolean("list_pending"); err == nil && listPending {
		txFilter.Status = common.CrossTxPendingStatus
	}
	if listCompleted, err := listTxsReq.GetFilterBoolean("list_completed"); err == nil && listCompleted {
		txFilter.Status = common.CrossTxCompletedStatus
	}
	if txHash, err := listTxsReq.GetFilterString("source_tx_hash"); err == nil && txHash != "" {
		txFilter.SourceTxHash = txHash
	}
	if txHash, err := listTxsReq.GetFilterString("dest_tx_hash"); err == nil && txHash != "" {
		txFilter.DestTxHash = sql.NullString{txHash, true}
	}
	txQuery := s.db.Preload("Chain").Preload("Reqs").Preload("Reqs.Asset").Where(txFilter)
	if onlyFromMainchain, err := listTxsReq.GetFilterBoolean("only_from_mainchain"); err == nil && onlyFromMainchain {
		txQuery = txQuery.Joins("join chains on chains.id = cross_transactions.chain_id").Where("chains.name = ?", common.MainchainName)
	}
	if onlyFromSidechain, err := listTxsReq.GetFilterBoolean("only_from_sidechain"); err == nil && onlyFromSidechain {
		txQuery = txQuery.Joins("join chains on chains.id = cross_transactions.chain_id").Where("chains.name = ?", common.SidechainName)
	}
	txQuery = txQuery.Order(fmt.Sprintf("cross_transactions.source_block_height %s", listTxsReq.Sorter.Order))
	txQuery = txQuery.Order(fmt.Sprintf("cross_transactions.source_tx_index %s", listTxsReq.Sorter.Order))
	if err := txQuery.Offset(query.Start).Limit(query.Limit).Find(&ormTxs).Error; err != nil {
		return nil, errors.Wrap(err, "query txs")
	}

	return ormTxs, nil
}
