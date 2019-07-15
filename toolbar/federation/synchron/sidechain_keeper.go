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
	"github.com/vapor/toolbar/federation/common"
	"github.com/vapor/toolbar/federation/config"
	"github.com/vapor/toolbar/federation/database"
	"github.com/vapor/toolbar/federation/database/orm"
	"github.com/vapor/toolbar/federation/service"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

type sidechainKeeper struct {
	cfg        *config.Chain
	db         *gorm.DB
	node       *service.Node
	assetStore *database.AssetStore
	chainID    uint64
}

func NewSidechainKeeper(db *gorm.DB, assetStore *database.AssetStore, cfg *config.Config) *sidechainKeeper {
	chain := &orm.Chain{Name: common.VaporChainName}
	if err := db.Where(chain).First(chain).Error; err != nil {
		log.WithField("err", err).Fatal("fail on get chain info")
	}

	return &sidechainKeeper{
		cfg:        &cfg.Sidechain,
		db:         db,
		node:       service.NewNode(cfg.Sidechain.Upstream),
		assetStore: assetStore,
		chainID:    chain.ID,
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

func (s *sidechainKeeper) createCrossChainReqs(db *gorm.DB, crossTransactionID uint64, tx *types.Tx, statusFail bool) error {
	fromAddress := common.ProgToAddress(tx.Inputs[0].ControlProgram(), &consensus.MainNetParams)
	for i, rawOutput := range tx.Outputs {
		if rawOutput.OutputType() != types.CrossChainOutputType {
			continue
		}

		if statusFail && *rawOutput.OutputCommitment().AssetAmount.AssetId != *consensus.BTMAssetID {
			continue
		}

		asset, err := s.assetStore.GetByAssetID(rawOutput.OutputCommitment().AssetAmount.AssetId.String())
		if err != nil {
			return err
		}

		prog := rawOutput.ControlProgram()
		req := &orm.CrossTransactionReq{
			CrossTransactionID: crossTransactionID,
			SourcePos:          uint64(i),
			AssetID:            asset.ID,
			AssetAmount:        rawOutput.OutputCommitment().AssetAmount.Amount,
			Script:             hex.EncodeToString(prog),
			FromAddress:        fromAddress,
			ToAddress:          common.ProgToAddress(prog, &consensus.BytomMainNetParams),
		}

		if err := db.Create(req).Error; err != nil {
			return err
		}
	}
	return nil
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

func (s *sidechainKeeper) processBlock(db *gorm.DB, block *types.Block, txStatus *bc.TransactionStatus) error {
	for i, tx := range block.Transactions {
		if s.isDepositTx(tx) {
			if err := s.processDepositTx(db, block, i); err != nil {
				return err
			}
		}

		if s.isWithdrawalTx(tx) {
			if err := s.processWithdrawalTx(db, block, txStatus, i); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *sidechainKeeper) processChainInfo(db *gorm.DB, block *types.Block) error {
	blockHash := block.Hash()
	res := db.Model(&orm.Chain{}).Where("block_hash = ?", block.PreviousBlockHash.String()).Updates(&orm.Chain{
		BlockHash:   blockHash.String(),
		BlockHeight: block.Height,
	})
	if err := res.Error; err != nil {
		return err
	}

	if res.RowsAffected != 1 {
		return ErrInconsistentDB
	}

	return nil
}

func (s *sidechainKeeper) processDepositTx(db *gorm.DB, block *types.Block, txIndex int) error {
	tx := block.Transactions[txIndex]
	sourceTxHash, err := s.locateMainChainTx(tx.Inputs[0])
	if err != nil {
		return err
	}

	blockHash := block.Hash()
	stmt := db.Model(&orm.CrossTransaction{}).Where(&orm.CrossTransaction{
		SourceTxHash: sourceTxHash,
		Status:       common.CrossTxPendingStatus,
	}).UpdateColumn(&orm.CrossTransaction{
		DestBlockHeight:    sql.NullInt64{int64(block.Height), true},
		DestBlockTimestamp: sql.NullInt64{int64(block.Timestamp / 1000), true},
		DestBlockHash:      sql.NullString{blockHash.String(), true},
		DestTxIndex:        sql.NullInt64{int64(txIndex), true},
		DestTxHash:         sql.NullString{tx.ID.String(), true},
		Status:             common.CrossTxCompletedStatus,
	})
	if stmt.Error != nil {
		return stmt.Error
	}

	if stmt.RowsAffected != 1 {
		return errors.Wrap(ErrInconsistentDB, "fail on find deposit data on database")
	}
	return nil
}

func (s *sidechainKeeper) processWithdrawalTx(db *gorm.DB, block *types.Block, txStatus *bc.TransactionStatus, txIndex int) error {
	tx := block.Transactions[txIndex]
	var muxID bc.Hash
	switch res := tx.Entries[*tx.ResultIds[0]].(type) {
	case *bc.CrossChainOutput:
		muxID = *res.Source.Ref
	case *bc.IntraChainOutput:
		muxID = *res.Source.Ref
	case *bc.VoteOutput:
		muxID = *res.Source.Ref
	case *bc.Retirement:
		muxID = *res.Source.Ref
	default:
		return ErrOutputType
	}

	rawTx, err := tx.MarshalText()
	if err != nil {
		return err
	}

	blockHash := block.Hash()
	ormTx := &orm.CrossTransaction{
		ChainID:              s.chainID,
		SourceBlockHeight:    block.Height,
		SourceBlockTimestamp: block.Timestamp / 1000,
		SourceBlockHash:      blockHash.String(),
		SourceTxIndex:        uint64(txIndex),
		SourceMuxID:          muxID.String(),
		SourceTxHash:         tx.ID.String(),
		SourceRawTransaction: string(rawTx),
		DestBlockHeight:      sql.NullInt64{Valid: false},
		DestBlockTimestamp:   sql.NullInt64{Valid: false},
		DestBlockHash:        sql.NullString{Valid: false},
		DestTxIndex:          sql.NullInt64{Valid: false},
		DestTxHash:           sql.NullString{Valid: false},
		Status:               common.CrossTxPendingStatus,
	}
	if err := db.Create(ormTx).Error; err != nil {
		return errors.Wrap(err, fmt.Sprintf("create sidechain WithdrawalTx %s", tx.ID.String()))
	}

	return s.createCrossChainReqs(db, ormTx.ID, tx, txStatus.VerifyStatus[txIndex].StatusFail)
}

func (s *sidechainKeeper) locateMainChainTx(input *types.TxInput) (string, error) {
	if input.InputType() != types.CrossChainInputType {
		return "", errors.New("found weird crossChain tx")
	}

	crossIn := input.TypedInput.(*types.CrossChainInput)
	crossTx := &orm.CrossTransaction{SourceMuxID: crossIn.SpendCommitment.SourceID.String()}
	if err := s.db.Where(crossTx).First(crossTx).Error; err != nil {
		return "", errors.Wrap(err, "fail on find CrossTransaction")
	}
	return crossTx.SourceTxHash, nil
}

func (s *sidechainKeeper) syncBlock() (bool, error) {
	chain := &orm.Chain{ID: s.chainID}
	if err := s.db.First(chain).Error; err != nil {
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
			"remote previous_block_Hash": nextBlock.PreviousBlockHash.String(),
			"db block_hash":              chain.BlockHash,
		}).Fatal("fail on block hash mismatch")
	}

	if err := s.tryAttachBlock(nextBlock, txStatus); err != nil {
		return false, err
	}

	return true, nil
}

func (s *sidechainKeeper) tryAttachBlock(block *types.Block, txStatus *bc.TransactionStatus) error {
	blockHash := block.Hash()
	log.WithFields(log.Fields{"block_height": block.Height, "block_hash": blockHash.String()}).Info("start to attachBlock")

	dbTx := s.db.Begin()
	if err := s.processBlock(dbTx, block, txStatus); err != nil {
		dbTx.Rollback()
		return err
	}

	if err := s.processChainInfo(dbTx, block); err != nil {
		dbTx.Rollback()
		return err
	}
	return dbTx.Commit().Error
}
