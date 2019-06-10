package synchron

import (
	"time"

	btmTypes "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/errors"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database/orm"
	"github.com/vapor/federation/service"
	"github.com/vapor/protocol/bc"
)

type mainchainKeeper struct {
	cfg       *config.Chain
	db        *gorm.DB
	node      *service.Node
	chainName string
}

func NewMainchainKeeper(db *gorm.DB, chainCfg *config.Chain) *mainchainKeeper {
	return &mainchainKeeper{
		cfg:       chainCfg,
		db:        db,
		node:      service.NewNode(chainCfg.Upstream),
		chainName: chainCfg.Name,
	}
}

func (m *mainchainKeeper) Run() {
	ticker := time.NewTicker(time.Duration(m.cfg.SyncSeconds) * time.Second)
	for ; true; <-ticker.C {
		for {
			isUpdate, err := m.syncBlock()
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

func (m *mainchainKeeper) syncBlock() (bool, error) {
	chain := &orm.Chain{Name: m.chainName}
	if err := m.db.Where(chain).First(chain).Error; err != nil {
		return false, errors.Wrap(err, "query chain")
	}

	height, err := m.node.GetBlockCount()
	if err != nil {
		return false, err
	}

	if height <= chain.BlockHeight+m.cfg.Confirmations {
		return false, nil
	}

	nextBlockStr, txStatus, err := m.node.GetBlockByHeight(chain.BlockHeight + 1)
	if err != nil {
		return false, err
	}

	nextBlock := &btmTypes.Block{}
	nextBlock.UnmarshalText([]byte(nextBlockStr))
	if nextBlock.PreviousBlockHash.String() == chain.BlockHash {
		return true, m.attachBlock(chain, nextBlock, txStatus)
	} else {
		log.WithFields(log.Fields{
			"remote PreviousBlockHash": nextBlock.PreviousBlockHash.String(),
			"db block_hash":            chain.BlockHash,
		}).Fatalf("PreviousBlockHash mismatch")
		return false, nil
	}
}

func (m *mainchainKeeper) attachBlock(chain *orm.Chain, block *btmTypes.Block, txStatus *bc.TransactionStatus) error {
	blockHash := block.Hash()
	log.WithFields(log.Fields{"block_height": block.Height, "block_hash": blockHash.String()}).Info("start to attachBlock")
	m.db.Begin()
	if err := m.processBlock(block); err != nil {
		m.db.Rollback()
		return err
	}

	return m.db.Commit().Error
}

func (m *mainchainKeeper) processBlock(block *btmTypes.Block) error {
	if err := m.processIssuing(block.Transactions); err != nil {
		return err
	}

	return nil
}

func (m *mainchainKeeper) processIssuing(txs []*btmTypes.Tx) error {
	return nil
	// if err := m.processIssuing(block.Transactions); err != nil {
	// 	return err
	// }
}
