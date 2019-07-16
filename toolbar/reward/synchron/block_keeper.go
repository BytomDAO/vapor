package synchron

import (
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/toolbar/common/service"
	"github.com/vapor/toolbar/reward/config"
	"github.com/vapor/toolbar/reward/database/orm"
)

type chainKeeper struct {
	cfg  *config.Chain
	db   *gorm.DB
	node *service.Node
}

func NewChainKeeper(db *gorm.DB, cfg *config.Config) *chainKeeper {
	return &chainKeeper{
		cfg:  &cfg.Chain,
		db:   db,
		node: service.NewNode(cfg.Chain.Upstream),
	}
}

func (c *chainKeeper) Run() {
	ticker := time.NewTicker(time.Duration(c.cfg.SyncSeconds) * time.Second)
	for ; true; <-ticker.C {
		for {
			isUpdate, err := c.syncBlock()
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

func (c *chainKeeper) syncBlock() (bool, error) {
	blockState := &orm.BlockState{}
	if err := c.db.First(blockState).Error; err != nil {
		return false, errors.Wrap(err, "query chain")
	}

	height, err := c.node.GetBlockCount()
	if err != nil {
		return false, err
	}

	if height == blockState.Height {
		return false, nil
	}

	nextBlockStr, txStatus, err := c.node.GetBlockByHeight(blockState.Height + 1)
	if err != nil {
		return false, err
	}

	nextBlock := &types.Block{}
	if err := nextBlock.UnmarshalText([]byte(nextBlockStr)); err != nil {
		return false, errors.New("Unmarshal nextBlock")
	}

	// Normal case, the previous hash of next block equals to the hash of current block,
	// just sync to database directly.
	if nextBlock.PreviousBlockHash.String() == blockState.BlockHash {
		return true, c.AttachBlock(nextBlock, txStatus)
	}

	log.WithField("block height", blockState.Height).Debug("the prev hash of remote is not equals the hash of current best block, must rollback")
	currentBlockStr, txStatus, err := c.node.GetBlockByHash(blockState.BlockHash)
	if err != nil {
		return false, err
	}

	currentBlock := &types.Block{}
	if err := nextBlock.UnmarshalText([]byte(currentBlockStr)); err != nil {
		return false, errors.New("Unmarshal currentBlock")
	}

	return true, c.DetachBlock(currentBlock, txStatus)
}

func (c *chainKeeper) AttachBlock(block *types.Block, txStatus *bc.TransactionStatus) error {
	return nil
}

func (c *chainKeeper) DetachBlock(block *types.Block, txStatus *bc.TransactionStatus) error {
	return nil
}
