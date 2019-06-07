package synchron

import (
	"time"

	// btmBc "github.com/bytom/protocol/bc"
	btmTypes "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/errors"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database/orm"
	"github.com/vapor/federation/service"
	// vaporBc "github.com/vapor/protocol/bc"
	vaporTypes "github.com/vapor/protocol/bc/types"
)

type blockKeeper struct {
	cfg       *config.Chain
	db        *gorm.DB
	node      *service.Node
	chainName string
}

func NewBlockKeeper(db *gorm.DB, chainCfg *config.Chain) *blockKeeper {
	return &blockKeeper{
		cfg:       chainCfg,
		db:        db,
		node:      service.NewNode(chainCfg.Upstream),
		chainName: chainCfg.Name,
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
	chain := &orm.Chain{Name: b.chainName}
	if err := b.db.Where(chain).First(chain).Error; err != nil {
		return false, errors.Wrap(err, "query chain")
	}

	height, err := b.node.GetBlockCount()
	if err != nil {
		return false, err
	}

	if height == chain.BlockHeight {
		return false, nil
	}

	nextBlockStr, txStatus, err := b.node.GetBlockByHeight(chain.BlockHeight + 1)
	if err != nil {
		return false, err
	}

	// Normal case, the previous hash of next block equals to the hash of current block,
	// just sync to database directly.
	switch {
	case b.cfg.IsMainchain:
		nextBlock := &btmTypes.Block{}
		nextBlock.UnmarshalText([]byte(nextBlockStr))
		if nextBlock.PreviousBlockHash.String() == chain.BlockHash {
			return true, b.AttachBlock(chain, nextBlock, txStatus)
		}

	default:
		nextBlock := &vaporTypes.Block{}
		nextBlock.UnmarshalText([]byte(nextBlockStr))
		if nextBlock.PreviousBlockHash.String() == chain.BlockHash {
			return true, b.AttachBlock(chain, nextBlock, txStatus)
		}
	}

	log.WithField("block height", chain.BlockHeight).Debug("the prev hash of remote is not equals the hash of current best block, must rollback")
	currentBlock, txStatus, err := b.node.GetBlockByHash(chain.BlockHash)
	if err != nil {
		return false, err
	}

	return true, b.DetachBlock(chain, currentBlock, txStatus)
}

func (b *blockKeeper) AttachBlock(chain *orm.Chain, block interface{}, txStatus interface{}) error {
	var blockHeight uint64
	var blockHashStr string
	switch {
	case b.cfg.IsMainchain:
		blockHeight = block.(*btmTypes.Block).Height
		blockHash := block.(*btmTypes.Block).Hash()
		blockHashStr = blockHash.String()
	default:
		blockHeight = block.(*vaporTypes.Block).Height
		blockHash := block.(*vaporTypes.Block).Hash()
		blockHashStr = blockHash.String()
	}
	log.WithFields(log.Fields{"block_height": blockHeight, "block_hash": blockHashStr}).Info("start to attachBlock")

	tx := b.db.Begin()
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

	return tx.Commit().Error
}

func (b *blockKeeper) DetachBlock(chain *orm.Chain, block interface{}, txStatus interface{}) error {
	var blockHeight uint64
	var blockHashStr string
	switch {
	case b.cfg.IsMainchain:
		blockHeight = block.(*btmTypes.Block).Height
		blockHash := block.(*btmTypes.Block).Hash()
		blockHashStr = blockHash.String()
	default:
		blockHeight = block.(*vaporTypes.Block).Height
		blockHash := block.(*vaporTypes.Block).Hash()
		blockHashStr = blockHash.String()
	}
	log.WithFields(log.Fields{"block_height": blockHeight, "block_hash": blockHashStr}).Info("start to detachBlock")

	tx := b.db.Begin()
	bp := &detachBlockProcessor{
		cfg:   b.cfg,
		db:    tx,
		chain: chain,
		block: block,
		// txStatus: txStatus,
	}
	if err := updateBlock(tx, bp); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
