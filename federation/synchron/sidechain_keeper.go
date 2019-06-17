package synchron

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/consensus"
	"github.com/vapor/errors"
	"github.com/vapor/federation/common"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database"
	"github.com/vapor/federation/database/orm"
	"github.com/vapor/federation/service"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

type sidechainKeeper struct {
	cfg        *config.Chain
	db         *gorm.DB
	node       *service.Node
	chainName  string
	assetCache *database.AssetCache
}

func NewSidechainKeeper(db *gorm.DB, cfg *config.Config) *sidechainKeeper {
	return &sidechainKeeper{
		cfg:        &cfg.Sidechain,
		db:         db,
		node:       service.NewNode(cfg.Sidechain.Upstream),
		chainName:  cfg.Sidechain.Name,
		assetCache: database.NewAssetCache(),
	}
}

func (s *sidechainKeeper) Run() {
	ticker := time.NewTicker(time.Duration(s.cfg.SyncSeconds) * time.Second)
	for ; true; <-ticker.C {
		for {
			isUpdate, err := s.syncBlock()
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

func (s *sidechainKeeper) syncBlock() (bool, error) {
	chain := &orm.Chain{Name: s.chainName}
	if err := s.db.Where(chain).First(chain).Error; err != nil {
		return false, errors.Wrap(err, "query chain")
	}

	height, err := s.node.GetBlockCount()
	if err != nil {
		return false, err
	}

	if height <= chain.BlockHeight+s.cfg.Confirmations {
		return false, nil
	}

	nextBlockStr, txStatus, err := s.node.GetBlockByHeight(chain.BlockHeight + 1)
	if err != nil {
		return false, err
	}

	nextBlock := &types.Block{}
	if err := nextBlock.UnmarshalText([]byte(nextBlockStr)); err != nil {
		return false, errors.New("Unmarshal nextBlock")
	}

	if nextBlock.PreviousBlockHash.String() != chain.BlockHash {
		log.WithFields(log.Fields{
			"remote PreviousBlockHash": nextBlock.PreviousBlockHash.String(),
			"db block_hash":            chain.BlockHash,
		}).Fatal("BlockHash mismatch")
		return false, ErrInconsistentDB
	}

	if err := s.tryAttachBlock(chain, nextBlock, txStatus); err != nil {
		return false, err
	}

	return true, nil
}

func (s *sidechainKeeper) tryAttachBlock(chain *orm.Chain, block *types.Block, txStatus *bc.TransactionStatus) error {
	blockHash := block.Hash()
	log.WithFields(log.Fields{"block_height": block.Height, "block_hash": blockHash.String()}).Info("start to attachBlock")
	s.db.Begin()
	if err := s.processBlock(chain, block, txStatus); err != nil {
		s.db.Rollback()
		return err
	}

	return s.db.Commit().Error
}

func (s *sidechainKeeper) processBlock(chain *orm.Chain, block *types.Block, txStatus *bc.TransactionStatus) error {
	for i, tx := range block.Transactions {
		if s.isDepositTx(tx) {
			if err := s.processDepositTx(chain, block, uint64(i), tx); err != nil {
				return err
			}
		}

		if s.isWithdrawalTx(tx) {
			if err := s.processWithdrawalTx(chain, block, txStatus, uint64(i), tx); err != nil {
				return err
			}
		}
	}

	return s.processChainInfo(chain, block)
}

func (s *sidechainKeeper) isDepositTx(tx *types.Tx) bool {
	for _, input := range tx.Inputs {
		if input.InputType() == types.CrossChainInputType {
			return true
		}
	}
	return false
}

func (s *sidechainKeeper) isWithdrawalTx(tx *types.Tx) bool {
	for _, output := range tx.Outputs {
		if output.OutputType() == types.CrossChainOutputType {
			return true
		}
	}
	return false
}

func (s *sidechainKeeper) processDepositTx(chain *orm.Chain, block *types.Block, txIndex uint64, tx *types.Tx) error {
	blockHash := block.Hash()
	stmt := s.db.Model(&orm.CrossTransaction{}).Where("chain_id != ?", chain.ID).
		Where(&orm.CrossTransaction{
			DestTxHash: sql.NullString{tx.ID.String(), true},
			Status:     common.CrossTxSubmittedStatus,
		}).UpdateColumn(&orm.CrossTransaction{
		DestBlockHeight: sql.NullInt64{int64(block.Height), true},
		DestBlockHash:   sql.NullString{blockHash.String(), true},
		DestTxIndex:     sql.NullInt64{int64(txIndex), true},
		Status:          common.CrossTxCompletedStatus,
	})
	if stmt.Error != nil {
		return stmt.Error
	}

	if stmt.RowsAffected != 1 {
		log.Warnf("sidechainKeeper.processDepositTx(%v): rows affected != 1", tx.ID.String())
	}
	return nil
}

func (s *sidechainKeeper) processWithdrawalTx(chain *orm.Chain, block *types.Block, txStatus *bc.TransactionStatus, txIndex uint64, tx *types.Tx) error {
	blockHash := block.Hash()

	var muxID bc.Hash
	res0ID := tx.ResultIds[0]
	switch res := tx.Entries[*res0ID].(type) {
	case *bc.CrossChainOutput:
		muxID = *res.Source.Ref
	case *bc.IntraChainOutput:
		muxID = *res.Source.Ref
	case *bc.VoteOutput:
		muxID = *res.Source.Ref
	default:
		return ErrOutputType
	}

	rawTx, err := tx.MarshalText()
	if err != nil {
		return err
	}

	ormTx := &orm.CrossTransaction{
		ChainID:              chain.ID,
		SourceBlockHeight:    block.Height,
		SourceBlockHash:      blockHash.String(),
		SourceTxIndex:        txIndex,
		SourceMuxID:          muxID.String(),
		SourceTxHash:         tx.ID.String(),
		SourceRawTransaction: string(rawTx),
		DestBlockHeight:      sql.NullInt64{Valid: false},
		DestBlockHash:        sql.NullString{Valid: false},
		DestTxIndex:          sql.NullInt64{Valid: false},
		DestTxHash:           sql.NullString{Valid: false},
		Status:               common.CrossTxPendingStatus,
	}
	if err := s.db.Create(ormTx).Error; err != nil {
		return errors.Wrap(err, fmt.Sprintf("create sidechain WithdrawalTx %s", tx.ID.String()))
	}

	statusFail := txStatus.VerifyStatus[txIndex].StatusFail
	crossChainOutputs, err := s.getCrossChainReqs(ormTx.ID, tx, statusFail)
	if err != nil {
		return err
	}

	for _, output := range crossChainOutputs {
		if err := s.db.Create(output).Error; err != nil {
			return errors.Wrap(err, fmt.Sprintf("create WithdrawalFromSidechain output: txid(%s), pos(%d)", tx.ID.String(), output.SourcePos))
		}
	}

	return nil
}

func (s *sidechainKeeper) getCrossChainReqs(crossTransactionID uint64, tx *types.Tx, statusFail bool) ([]*orm.CrossTransactionReq, error) {
	// assume inputs are from an identical owner
	script := hex.EncodeToString(tx.Inputs[0].ControlProgram())
	inputs := []*orm.CrossTransactionReq{}
	for i, rawOutput := range tx.Outputs {
		// check valid withdrawal
		if rawOutput.OutputType() != types.CrossChainOutputType {
			continue
		}

		if statusFail && *rawOutput.OutputCommitment().AssetAmount.AssetId != *consensus.BTMAssetID {
			continue
		}

		asset, err := s.getAsset(rawOutput.OutputCommitment().AssetAmount.AssetId.String())
		if err != nil {
			return nil, err
		}

		input := &orm.CrossTransactionReq{
			CrossTransactionID: crossTransactionID,
			SourcePos:          uint64(i),
			AssetID:            asset.ID,
			AssetAmount:        rawOutput.OutputCommitment().AssetAmount.Amount,
			Script:             script,
		}
		inputs = append(inputs, input)
	}
	return inputs, nil
}

func (s *sidechainKeeper) processChainInfo(chain *orm.Chain, block *types.Block) error {
	blockHash := block.Hash()
	chain.BlockHash = blockHash.String()
	chain.BlockHeight = block.Height
	res := s.db.Model(chain).Where("block_hash = ?", block.PreviousBlockHash.String()).Updates(chain)
	if err := res.Error; err != nil {
		return err
	}

	if res.RowsAffected != 1 {
		return ErrInconsistentDB
	}

	return nil
}

func (s *sidechainKeeper) getAsset(assetID string) (*orm.Asset, error) {
	if asset := s.assetCache.Get(assetID); asset != nil {
		return asset, nil
	}

	asset := &orm.Asset{AssetID: assetID}
	if err := s.db.Where(asset).First(asset).Error; err != nil {
		return nil, errors.Wrap(err, "asset not found in memory and mysql")
	}

	s.assetCache.Add(assetID, asset)
	return asset, nil
}
