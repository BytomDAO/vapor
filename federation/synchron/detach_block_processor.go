package synchron

import (
	// "math/big"
	// "github.com/bytom/protocol/bc"
	btmTypes "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"

	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database/orm"
)

type detachBlockProcessor struct {
	cfg   *config.Chain
	db    *gorm.DB
	chain *orm.Chain
	block interface{}
	// txStatus *bc.TransactionStatus
}

func (p *detachBlockProcessor) processIssuing(db *gorm.DB, txs []*btmTypes.Tx) error {
	return nil
}

func (p *detachBlockProcessor) processChainInfo() error {
	// p.coin.BlockHeight = p.block.Height - 1
	// p.coin.BlockHash = p.block.PreviousBlockHash.String()
	// oldBlockHash := p.block.Hash()
	// db := p.db.Model(p.coin).Where("block_hash = ?", oldBlockHash.String()).Updates(p.coin)
	// if err := db.Error; err != nil {
	// 	return err
	// }

	// if db.RowsAffected != 1 {
	// 	return ErrInconsistentDB
	// }
	return nil
}
