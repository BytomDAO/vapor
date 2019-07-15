package synchron

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	btmConsensus "github.com/bytom/consensus"
	btmBc "github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/vm"
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
)

type mainchainKeeper struct {
	cfg            *config.Chain
	db             *gorm.DB
	node           *service.Node
	assetStore     *database.AssetStore
	chainID        uint64
	federationProg []byte
}

func NewMainchainKeeper(db *gorm.DB, assetStore *database.AssetStore, cfg *config.Config) *mainchainKeeper {
	chain := &orm.Chain{Name: common.BytomChainName}
	if err := db.Where(chain).First(chain).Error; err != nil {
		log.WithField("err", err).Fatal("fail on get chain info")
	}

	return &mainchainKeeper{
		cfg:            &cfg.Mainchain,
		db:             db,
		node:           service.NewNode(cfg.Mainchain.Upstream),
		assetStore:     assetStore,
		federationProg: cfg.FederationProg,
		chainID:        chain.ID,
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

func (m *mainchainKeeper) createCrossChainReqs(db *gorm.DB, crossTransactionID uint64, tx *types.Tx, statusFail bool) error {
	prog := tx.Inputs[0].ControlProgram()
	fromAddress := common.ProgToAddress(prog, &consensus.BytomMainNetParams)
	toAddress := common.ProgToAddress(prog, &consensus.MainNetParams)
	for i, rawOutput := range tx.Outputs {
		if !bytes.Equal(rawOutput.OutputCommitment.ControlProgram, m.federationProg) {
			continue
		}

		if statusFail && *rawOutput.OutputCommitment.AssetAmount.AssetId != *btmConsensus.BTMAssetID {
			continue
		}

		asset, err := m.assetStore.GetByAssetID(rawOutput.OutputCommitment.AssetAmount.AssetId.String())
		if err != nil {
			return err
		}

		req := &orm.CrossTransactionReq{
			CrossTransactionID: crossTransactionID,
			SourcePos:          uint64(i),
			AssetID:            asset.ID,
			AssetAmount:        rawOutput.OutputCommitment.AssetAmount.Amount,
			Script:             hex.EncodeToString(prog),
			FromAddress:        fromAddress,
			ToAddress:          toAddress,
		}

		if err := db.Create(req).Error; err != nil {
			return err
		}
	}
	return nil
}

func (m *mainchainKeeper) isDepositTx(tx *types.Tx) bool {
	for _, input := range tx.Inputs {
		if bytes.Equal(input.ControlProgram(), m.federationProg) {
			return false
		}
	}

	for _, output := range tx.Outputs {
		if bytes.Equal(output.OutputCommitment.ControlProgram, m.federationProg) {
			return true
		}
	}
	return false
}

func (m *mainchainKeeper) isWithdrawalTx(tx *types.Tx) bool {
	for _, input := range tx.Inputs {
		if !bytes.Equal(input.ControlProgram(), m.federationProg) {
			return false
		}
	}

	if sourceTxHash := locateSideChainTx(tx.Outputs[len(tx.Outputs)-1]); sourceTxHash == "" {
		return false
	}
	return false
}

func locateSideChainTx(output *types.TxOutput) string {
	insts, err := vm.ParseProgram(output.OutputCommitment.ControlProgram)
	if err != nil {
		return ""
	}

	if len(insts) != 2 {
		return ""
	}

	return hex.EncodeToString(insts[1].Data)
}

func (m *mainchainKeeper) processBlock(db *gorm.DB, block *types.Block, txStatus *bc.TransactionStatus) error {
	for i, tx := range block.Transactions {
		if err := m.processIssuance(tx); err != nil {
			return err
		}

		if m.isDepositTx(tx) {
			if err := m.processDepositTx(db, block, txStatus, i); err != nil {
				return err
			}
		}

		if m.isWithdrawalTx(tx) {
			if err := m.processWithdrawalTx(db, block, i); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *mainchainKeeper) processChainInfo(db *gorm.DB, block *types.Block) error {
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

func (m *mainchainKeeper) processDepositTx(db *gorm.DB, block *types.Block, txStatus *bc.TransactionStatus, txIndex int) error {
	tx := block.Transactions[txIndex]
	var muxID btmBc.Hash
	switch res := tx.Entries[*tx.ResultIds[0]].(type) {
	case *btmBc.Output:
		muxID = *res.Source.Ref
	case *btmBc.Retirement:
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
		ChainID:              m.chainID,
		SourceBlockHeight:    block.Height,
		SourceBlockTimestamp: block.Timestamp,
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
		return errors.Wrap(err, fmt.Sprintf("create mainchain DepositTx %s", tx.ID.String()))
	}

	return m.createCrossChainReqs(db, ormTx.ID, tx, txStatus.VerifyStatus[txIndex].StatusFail)
}

func (m *mainchainKeeper) processIssuance(tx *types.Tx) error {
	for _, input := range tx.Inputs {
		if input.InputType() != types.IssuanceInputType {
			continue
		}

		issuance := input.TypedInput.(*types.IssuanceInput)
		assetID := issuance.AssetID()
		if _, err := m.assetStore.GetByAssetID(assetID.String()); err == nil {
			continue
		}

		asset := &orm.Asset{
			AssetID:         assetID.String(),
			IssuanceProgram: hex.EncodeToString(issuance.IssuanceProgram),
			VMVersion:       issuance.VMVersion,
			Definition:      string(issuance.AssetDefinition),
		}

		if err := m.db.Create(asset).Error; err != nil {
			return err
		}
	}
	return nil
}

func (m *mainchainKeeper) processWithdrawalTx(db *gorm.DB, block *types.Block, txIndex int) error {
	blockHash := block.Hash()
	tx := block.Transactions[txIndex]

	stmt := db.Model(&orm.CrossTransaction{}).Where(&orm.CrossTransaction{
		SourceTxHash: locateSideChainTx(tx.Outputs[len(tx.Outputs)-1]),
		Status:       common.CrossTxPendingStatus,
	}).UpdateColumn(&orm.CrossTransaction{
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

func (m *mainchainKeeper) syncBlock() (bool, error) {
	chain := &orm.Chain{ID: m.chainID}
	if err := m.db.First(chain).Error; err != nil {
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

	if err := m.tryAttachBlock(nextBlock, txStatus); err != nil {
		return false, err
	}

	return true, nil
}

func (m *mainchainKeeper) tryAttachBlock(block *types.Block, txStatus *bc.TransactionStatus) error {
	blockHash := block.Hash()
	log.WithFields(log.Fields{"block_height": block.Height, "block_hash": blockHash.String()}).Info("start to attachBlock")

	dbTx := m.db.Begin()
	if err := m.processBlock(dbTx, block, txStatus); err != nil {
		dbTx.Rollback()
		return err
	}

	if err := m.processChainInfo(dbTx, block); err != nil {
		dbTx.Rollback()
		return err
	}

	return dbTx.Commit().Error
}
