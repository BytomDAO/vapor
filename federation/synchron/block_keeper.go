package synchron

import (
	"time"

	// "github.com/bytom/errors"
	// "github.com/bytom/protocol/bc"
	// "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/federation/config"
	// "github.com/blockcenter/database"
	// "github.com/blockcenter/database/orm"
	"github.com/vapor/federation/service"
)

type blockKeeper struct {
	cfg  *config.Chain
	db   *gorm.DB
	node *service.Node
	// cache    *database.RedisDB
	// coinName string
}

func NewBlockKeeper(db *gorm.DB, chainCfg *config.Chain) *blockKeeper {
	return &blockKeeper{
		cfg:  chainCfg,
		db:   db,
		node: service.NewNode(chainCfg.Upstream),
	}
}

func (b *blockKeeper) Run() {
	ticker := time.NewTicker(time.Duration(b.cfg.SyncSeconds) * time.Second)
	for ; true; <-ticker.C {
		for {
			isUpdate, err := b.syncBlock()
			if err != nil {
				log.WithField("error", err).Errorln("blockKeeper fail on process block")
				break
			}

			if !isUpdate {
				break
			}
		}
	}
}

func (b *blockKeeper) syncBlock() (bool, error) {
	/*
		coin := &orm.Coin{Name: b.coinName}
		if err := b.db.Where(coin).First(coin).Error; err != nil {
			return false, errors.Wrap(err, "query coin")
		}

		height, err := b.node.GetBlockCount()
		if err != nil {
			return false, err
		}

		if height == coin.BlockHeight {
			return false, nil
		}

		nextBlock, txStatus, err := b.node.GetBlockByHeight(coin.BlockHeight + 1)
		if err != nil {
			return false, err
		}

		// Normal case, the previous hash of next block equals to the hash of current block,
		// just sync to database directly.
		if nextBlock.PreviousBlockHash.String() == coin.BlockHash {
			return true, b.AttachBlock(coin, nextBlock, txStatus)
		}

		log.WithField("block height", coin.BlockHeight).Debug("the prev hash of remote is not equals the hash of current best block, must rollback")
		currentBlock, txStatus, err := b.node.GetBlockByHash(coin.BlockHash)
		if err != nil {
			return false, err
		}
	*/
	// return true, b.DetachBlock( /*coin, */ currentBlock, txStatus)
	return true, nil
}

func (b *blockKeeper) AttachBlock(coin *orm.Coin, block *types.Block, txStatus *bc.TransactionStatus) error {
	// blockHash := block.Hash()
	// log.WithFields(log.Fields{"block_height": block.Height, "block_hash": blockHash.String()}).Info("start to attachBlock")

	// tx := b.db.Begin()
	// bp := &attachBlockProcessor{
	// 	db:       tx,
	// 	block:    block,
	// 	coin:     coin,
	// 	txStatus: txStatus,
	// }
	// if err := updateBlock(b.cfg, tx, bp); err != nil {
	// 	tx.Rollback()
	// 	return err
	// }

	// return tx.Commit().Error
	return nil
}

func (b *blockKeeper) DetachBlock(coin *orm.Coin, block *types.Block, txStatus *bc.TransactionStatus) error {
	// blockHash := block.Hash()
	// log.WithFields(log.Fields{"block_height": block.Height, "block_hash": blockHash.String()}).Info("start to detachBlock")

	// tx := b.db.Begin()
	// bp := &detachBlockProcessor{
	// 	db:       tx,
	// 	block:    block,
	// 	coin:     coin,
	// 	txStatus: txStatus,
	// }
	// if err := updateBlock(b.cfg, tx, bp); err != nil {
	// 	tx.Rollback()
	// 	return err
	// }

	// return tx.Commit().Error
	return nil
}
