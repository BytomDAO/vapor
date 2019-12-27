package api

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/toolbar/federation/common"
	"github.com/bytom/vapor/toolbar/federation/database/orm"
	serverCommon "github.com/bytom/vapor/toolbar/server"
)

type listCrosschainTxsReq struct{ serverCommon.Display }

func (s *Server) ListCrosschainTxs(c *gin.Context, listTxsReq *listCrosschainTxsReq, query *serverCommon.PaginationQuery) ([]*orm.CrossTransaction, error) {
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
		txFilter.DestTxHash = sql.NullString{String: txHash, Valid: true}
	}

	txQuery := s.db.Preload("Chain").Preload("Reqs").Preload("Reqs.Asset").Where(txFilter)
	// filter direction
	if sourceChainName, err := listTxsReq.GetFilterString("source_chain_name"); err == nil && sourceChainName != "" {
		txQuery = txQuery.Joins("join chains on chains.id = cross_transactions.chain_id").Where("chains.name = ?", sourceChainName)
	}

	// filter address
	if address, err := listTxsReq.GetFilterString("address"); err == nil && address != "" {
		txQuery = txQuery.Joins("join cross_transaction_reqs on cross_transaction_reqs.cross_transaction_id = cross_transactions.id").
			Where("cross_transaction_reqs.from_address = ? or cross_transaction_reqs.to_address = ?", address, address)
	}

	// sorter order
	txQuery = txQuery.Order(fmt.Sprintf("cross_transactions.source_block_height %s", listTxsReq.Sorter.Order))
	txQuery = txQuery.Order(fmt.Sprintf("cross_transactions.source_tx_index %s", listTxsReq.Sorter.Order))
	if err := txQuery.Offset(query.Start).Limit(query.Limit).Find(&ormTxs).Error; err != nil {
		return nil, errors.Wrap(err, "query txs")
	}

	return ormTxs, nil
}

func (s *Server) ListChains(c *gin.Context) ([]*orm.Chain, error) {
	var chains []*orm.Chain
	if err := s.db.Find(&chains).Error; err != nil {
		return nil, err
	}

	return chains, nil
}
