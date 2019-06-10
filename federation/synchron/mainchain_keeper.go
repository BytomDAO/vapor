package synchron

import (
	"encoding/hex"
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
	assetMap  map[string]*orm.Asset
}

func NewMainchainKeeper(db *gorm.DB, chainCfg *config.Chain) *mainchainKeeper {
	return &mainchainKeeper{
		cfg:       chainCfg,
		db:        db,
		node:      service.NewNode(chainCfg.Upstream),
		chainName: chainCfg.Name,
		assetMap:  make(map[string]*orm.Asset),
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
	if nextBlock.PreviousBlockHash.String() != chain.BlockHash {
		log.WithFields(log.Fields{
			"remote PreviousBlockHash": nextBlock.PreviousBlockHash.String(),
			"db block_hash":            chain.BlockHash,
		}).Fatal("BlockHash mismatch")
		return false, errors.New("BlockHash mismatch")
	}

	if err := m.tryAttachBlock(chain, nextBlock, txStatus); err != nil {
		return false, err
	}

	return true, nil
}

func (m *mainchainKeeper) tryAttachBlock(chain *orm.Chain, block *btmTypes.Block, txStatus *bc.TransactionStatus) error {
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
	var assets []*orm.Asset

	for _, tx := range txs {
		for _, input := range tx.Inputs {
			switch inp := input.TypedInput.(type) {
			case *btmTypes.IssuanceInput:
				assetID := inp.AssetID()
				if _, err := m.getAsset(assetID.String()); err == nil {
					continue
				}

				asset := &orm.Asset{
					AssetID:           assetID.String(),
					IssuanceProgram:   hex.EncodeToString(inp.IssuanceProgram),
					VMVersion:         inp.VMVersion,
					RawDefinitionByte: hex.EncodeToString(inp.AssetDefinition),
				}
				assets = append(assets, asset)
			}
		}
	}

	for _, asset := range assets {
		if err := m.db.Create(asset).Error; err != nil {
			return err
		}

		m.assetMap[asset.AssetID] = asset
	}

	return nil
}

func (m *mainchainKeeper) getAsset(assetID string) (*orm.Asset, error) {
	asset, ok := m.assetMap[assetID]
	if ok {
		return asset, nil
	}

	asset = &orm.Asset{AssetID: assetID}
	if err := m.db.Where(asset).First(asset).Error; err != nil {
		return nil, errors.Wrap(err, "asset not found in memory and mysql")
	}

	m.assetMap[assetID] = asset
	return asset, nil
}
