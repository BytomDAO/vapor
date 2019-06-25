package api

import (
	"github.com/gin-gonic/gin"

	"github.com/vapor/federation/database/orm"
)

func (s *Server) ListCrosschainTxs(c *gin.Context, req *listCrosschainTxsReq, query *PaginationQuery) ([]*orm.CrossTransaction, error) {
	var ormTxs []*orm.CrossTransaction

	// addressFilter := &orm.Address{WalletID: sql.NullInt64{Int64: wallet.ID, Valid: true}}
	// txQuery := s.db.Slave().Joins("join address_transactions on transactions.id = address_transactions.transaction_id")
	// txQuery = txQuery.Joins("join addresses on addresses.id = address_transactions.address_id")
	// txQuery = txQuery.Group("address_transactions.transaction_id").Where(addressFilter).Preload("Memo", "wallet_id = ?", wallet.ID)

	// if asset, err := listTxsReq.GetFilterString("asset_id"); err == nil && asset != "" {
	// 	txQuery = txQuery.Joins("join assets on assets.id = address_transactions.asset_id").Where(&orm.Asset{Asset: asset})
	// }
	// if txHash, err := listTxsReq.GetFilterString("tx_hash"); err == nil && txHash != "" {
	// 	txQuery = txQuery.Where("hash = ?", txHash)
	// }
	// if listTxsReq.Sorter.By == "amount" {
	// 	txQuery = txQuery.Order(fmt.Sprintf("address_transactions.amount %s", listTxsReq.Sorter.Order))
	// }
	// txQuery = txQuery.Order(fmt.Sprintf("address_transactions.transaction_id %s", listTxsReq.Sorter.Order))
	// if err := txQuery.Offset(query.Start).Limit(query.Limit).Find(&ormTxs).Error; err != nil {
	// 	return nil, errors.Wrap(err, "query txs")
	// }

	return ormTxs, nil
}
