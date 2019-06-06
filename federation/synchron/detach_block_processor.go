package synchron

import (
	// "math/big"
	// "github.com/bytom/protocol/bc"
	btmTypes "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"
	// "github.com/blockcenter/database/orm"
)

type detachBlockProcessor struct {
	db *gorm.DB
	// coin     *orm.Coin
	// block    *btmTypes.Block
	// txStatus *bc.TransactionStatus
}

func (p *detachBlockProcessor) processIssuing(db *gorm.DB, txs []*btmTypes.Tx) error {
	return nil
}
