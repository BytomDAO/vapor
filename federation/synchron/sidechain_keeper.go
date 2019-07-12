package synchron

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/consensus"
	"github.com/vapor/consensus/segwit"
	"github.com/vapor/errors"
	"github.com/vapor/federation/common"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database"
	"github.com/vapor/federation/database/orm"
	"github.com/vapor/federation/service"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/wallet"
)

type sidechainKeeper struct {
	cfg        *config.Chain
	db         *gorm.DB
	node       *service.Node
	assetStore *database.AssetStore
}

func NewSidechainKeeper(db *gorm.DB, assetStore *database.AssetStore, cfg *config.Config) *sidechainKeeper {
	return &sidechainKeeper{
		cfg:        &cfg.Sidechain,
		db:         db,
		node:       service.NewNode(cfg.Sidechain.Upstream),
		assetStore: assetStore,
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
	chain := &orm.Chain{Name: common.VaporChainName}
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

	dbTx := s.db.Begin()
	if err := s.processBlock(dbTx, chain, block, txStatus); err != nil {
		dbTx.Rollback()
		return err
	}

	if err := s.processChainInfo(dbTx, chain, block); err != nil {
		dbTx.Rollback()
		return err
	}
	return dbTx.Commit().Error
}

func (s *sidechainKeeper) processBlock(db *gorm.DB, chain *orm.Chain, block *types.Block, txStatus *bc.TransactionStatus) error {
	for i, tx := range block.Transactions {
		if s.isDepositTx(tx) {
			if err := s.processDepositTx(db, chain, block, uint64(i), tx); err != nil {
				return err
			}
		}

		if s.isWithdrawalTx(tx) {
			if err := s.processWithdrawalTx(db, chain, block, txStatus, uint64(i), tx); err != nil {
				return err
			}
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

func (s *sidechainKeeper) processDepositTx(db *gorm.DB, chain *orm.Chain, block *types.Block, txIndex uint64, tx *types.Tx) error {
	sourceTxHash, err := s.locateMainChainTx(tx.Inputs[0])
	if err != nil {
		return err
	}

	blockHash := block.Hash()
	stmt := db.Model(&orm.CrossTransaction{}).Where("source_tx_hash = ? ", sourceTxHash).UpdateColumn(&orm.CrossTransaction{
		DestBlockHeight:    sql.NullInt64{int64(block.Height), true},
		DestBlockTimestamp: sql.NullInt64{int64(block.Timestamp), true},
		DestBlockHash:      sql.NullString{blockHash.String(), true},
		DestTxIndex:        sql.NullInt64{int64(txIndex), true},
		DestTxHash:         sql.NullString{tx.ID.String(), true},
		Status:             common.CrossTxCompletedStatus,
	})
	if stmt.Error != nil {
		return stmt.Error
	}

	if stmt.RowsAffected != 1 {
		return ErrInconsistentDB
	}
	return nil
}

func (s *sidechainKeeper) processWithdrawalTx(db *gorm.DB, chain *orm.Chain, block *types.Block, txStatus *bc.TransactionStatus, txIndex uint64, tx *types.Tx) error {
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

	blockHash := block.Hash()
	ormTx := &orm.CrossTransaction{
		ChainID:              chain.ID,
		SourceBlockHeight:    block.Height,
		SourceBlockTimestamp: block.Timestamp,
		SourceBlockHash:      blockHash.String(),
		SourceTxIndex:        txIndex,
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

func (s *sidechainKeeper) createCrossChainReqs(db *gorm.DB, crossTransactionID uint64, tx *types.Tx, statusFail bool) error {
	var fromAddress string
	inputCP := tx.Inputs[0].ControlProgram()
	switch {
	case segwit.IsP2WPKHScript(inputCP):
		if pubHash, err := segwit.GetHashFromStandardProg(inputCP); err == nil {
			fromAddress = wallet.BuildP2PKHAddress(pubHash, &consensus.MainNetParams)
		}
	case segwit.IsP2WSHScript(inputCP):
		if scriptHash, err := segwit.GetHashFromStandardProg(inputCP); err == nil {
			fromAddress = wallet.BuildP2SHAddress(scriptHash, &consensus.MainNetParams)
		}
	}

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

		var toAddress string
		outputCP := rawOutput.ControlProgram()
		switch {
		case segwit.IsP2WPKHScript(outputCP):
			if pubHash, err := segwit.GetHashFromStandardProg(outputCP); err == nil {
				toAddress = wallet.BuildP2PKHAddress(pubHash, &consensus.BytomMainNetParams)
			}
		case segwit.IsP2WSHScript(outputCP):
			if scriptHash, err := segwit.GetHashFromStandardProg(outputCP); err == nil {
				toAddress = wallet.BuildP2SHAddress(scriptHash, &consensus.BytomMainNetParams)
			}
		}

		req := &orm.CrossTransactionReq{
			CrossTransactionID: crossTransactionID,
			SourcePos:          uint64(i),
			AssetID:            asset.ID,
			AssetAmount:        rawOutput.OutputCommitment().AssetAmount.Amount,
			Script:             hex.EncodeToString(rawOutput.ControlProgram()),
			FromAddress:        fromAddress,
			ToAddress:          toAddress,
		}

		if err := db.Create(req).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *sidechainKeeper) processChainInfo(db *gorm.DB, chain *orm.Chain, block *types.Block) error {
	blockHash := block.Hash()
	chain.BlockHash = blockHash.String()
	chain.BlockHeight = block.Height
	res := db.Model(chain).Where("block_hash = ?", block.PreviousBlockHash.String()).Updates(chain)
	if err := res.Error; err != nil {
		return err
	}

	if res.RowsAffected != 1 {
		return ErrInconsistentDB
	}

	return nil
}

func (s *sidechainKeeper) locateMainChainTx(input *types.TxInput) (string, error) {
	if input.InputType() != types.CrossChainInputType {
		return "", errors.New("found weird crossChain tx")
	}

	crossIn := input.TypedInput.(*types.CrossChainInput)
	crossTx := &orm.CrossTransaction{SourceMuxID: crossIn.SpendCommitment.SourceID.String()}
	if err := s.
		db.Where(crossTx).First(crossTx).Error; err != nil {
		return "", errors.Wrap(err, "fail on find CrossTransaction")
	}
	return crossTx.SourceTxHash, nil
}
