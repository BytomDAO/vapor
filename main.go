package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"

	"github.com/vapor/federation/common"
	// "github.com/vapor/federation/config"
	"github.com/vapor/federation/database/orm"
)

func main() {
	dsnTemplate := "%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=true&loc=Local"
	dsn := fmt.Sprintf(dsnTemplate, "root", "toor", "127.0.0.1", 3306, "federation")
	db, err := gorm.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	db.LogMode(true)
	// UPDATE `cross_transactions` SET `dest_block_hash` = 'blockHash.String()', `dest_block_height` = '2', `dest_tx_index` = '3', `status` = '3'  WHERE (`cross_transactions`.`chain_id` = '1') AND (`cross_transactions`.`dest_tx_hash` = 'tx.ID.String()') AND (`cross_transactions`.`status` = '2')
	if err := db.Model(&orm.CrossTransaction{}).Where("chain_id != ?", 1).
		Where(&orm.CrossTransaction{
			DestTxHash: sql.NullString{"tx.ID.String()", true},
			Status:     common.CrossTxSubmittedStatus,
		}).UpdateColumn(&orm.CrossTransaction{
		DestBlockHeight: sql.NullInt64{int64(2), true},
		DestBlockHash:   sql.NullString{"blockHash.String()", true},
		DestTxIndex:     sql.NullInt64{int64(3), true},
		Status:          common.CrossTxCompletedStatus,
	}).Error; err != nil {
		panic(err)
	}
}
