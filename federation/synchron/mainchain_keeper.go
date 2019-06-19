package synchron

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bytom/consensus"
	btmBc "github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/errors"
	"github.com/vapor/federation/common"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database"
	"github.com/vapor/federation/database/orm"
	"github.com/vapor/federation/service"
	"github.com/vapor/federation/util"
	"github.com/vapor/protocol/bc"
)

type mainchainKeeper struct {
	cfg        *config.Chain
	db         *gorm.DB
	node       *service.Node
	chainName  string
	assetStore *database.AssetStore
	fedProg    []byte
}

func NewMainchainKeeper(db *gorm.DB, assetStore *database.AssetStore, cfg *config.Config) *mainchainKeeper {
	return &mainchainKeeper{
		cfg:        &cfg.Mainchain,
		db:         db,
		node:       service.NewNode(cfg.Mainchain.Upstream),
		chainName:  cfg.Mainchain.Name,
		assetStore: assetStore,
		fedProg:    util.SegWitWrap(util.ParseFedProg(cfg.Warders, cfg.Quorum)),
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

	if err := m.tryAttachBlock(chain, nextBlock, txStatus); err != nil {
		return false, err
	}

	return true, nil
}

func (m *mainchainKeeper) tryAttachBlock(chain *orm.Chain, block *types.Block, txStatus *bc.TransactionStatus) error {
	blockHash := block.Hash()
	log.WithFields(log.Fields{"block_height": block.Height, "block_hash": blockHash.String()}).Info("start to attachBlock")
	dbTx := m.db.Begin()
	if err := m.processBlock(chain, block, txStatus); err != nil {
		dbTx.Rollback()
		return err
	}

	return dbTx.Commit().Error
}

func (m *mainchainKeeper) processBlock(chain *orm.Chain, block *types.Block, txStatus *bc.TransactionStatus) error {
	if err := m.processIssuing(block.Transactions); err != nil {
		return err
	}

	for i, tx := range block.Transactions {
		if m.isDepositTx(tx) {
			if err := m.processDepositTx(chain, block, txStatus, uint64(i), tx); err != nil {
				return err
			}
		}

		if m.isWithdrawalTx(tx) {
			if err := m.processWithdrawalTx(chain, block, uint64(i), tx); err != nil {
				return err
			}
		}
	}

	return m.processChainInfo(chain, block)
}

func (m *mainchainKeeper) isDepositTx(tx *types.Tx) bool {
	for _, output := range tx.Outputs {
		if bytes.Equal(output.OutputCommitment.ControlProgram, m.fedProg) {
			return true
		}
	}
	return false
}

func (m *mainchainKeeper) isWithdrawalTx(tx *types.Tx) bool {
	for _, input := range tx.Inputs {
		if bytes.Equal(input.ControlProgram(), m.fedProg) {
			return true
		}
	}
	return false
}

func (m *mainchainKeeper) processDepositTx(chain *orm.Chain, block *types.Block, txStatus *bc.TransactionStatus, txIndex uint64, tx *types.Tx) error {
	blockHash := block.Hash()

	var muxID btmBc.Hash
	res0ID := tx.ResultIds[0]
	switch res := tx.Entries[*res0ID].(type) {
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
		Status:               common.CrossTxInitiatedStatus,
	}
	if err := m.db.Create(ormTx).Error; err != nil {
		return errors.Wrap(err, fmt.Sprintf("create mainchain DepositTx %s", tx.ID.String()))
	}

	statusFail := txStatus.VerifyStatus[txIndex].StatusFail
	crossChainInputs, err := m.getCrossChainReqs(ormTx.ID, tx, statusFail)
	if err != nil {
		return err
	}

	for _, input := range crossChainInputs {
		if err := m.db.Create(input).Error; err != nil {
			return errors.Wrap(err, fmt.Sprintf("create DepositFromMainchain input: txid(%s), pos(%d)", tx.ID.String(), input.SourcePos))
		}
	}

	return nil
}

func (m *mainchainKeeper) getCrossChainReqs(crossTransactionID uint64, tx *types.Tx, statusFail bool) ([]*orm.CrossTransactionReq, error) {
	// assume inputs are from an identical owner
	script := hex.EncodeToString(tx.Inputs[0].ControlProgram())
	reqs := []*orm.CrossTransactionReq{}
	for i, rawOutput := range tx.Outputs {
		// check valid deposit
		if !bytes.Equal(rawOutput.OutputCommitment.ControlProgram, m.fedProg) {
			continue
		}

		if statusFail && *rawOutput.OutputCommitment.AssetAmount.AssetId != *consensus.BTMAssetID {
			continue
		}

		asset, err := m.assetStore.GetByAssetID(rawOutput.OutputCommitment.AssetAmount.AssetId.String())
		if err != nil {
			return nil, err
		}

		req := &orm.CrossTransactionReq{
			CrossTransactionID: crossTransactionID,
			SourcePos:          uint64(i),
			AssetID:            asset.ID,
			AssetAmount:        rawOutput.OutputCommitment.AssetAmount.Amount,
			Script:             script,
		}
		reqs = append(reqs, req)
	}
	return reqs, nil
}

func (m *mainchainKeeper) processWithdrawalTx(chain *orm.Chain, block *types.Block, txIndex uint64, tx *types.Tx) error {
	blockHash := block.Hash()
	stmt := m.db.Model(&orm.CrossTransaction{}).Where("chain_id != ?", chain.ID).
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
		log.Warnf("mainchainKeeper.processWithdrawalTx(%v): rows affected != 1", tx.ID.String())
	}
	return nil
}

func (m *mainchainKeeper) processChainInfo(chain *orm.Chain, block *types.Block) error {
	blockHash := block.Hash()
	chain.BlockHash = blockHash.String()
	chain.BlockHeight = block.Height
	res := m.db.Model(chain).Where("block_hash = ?", block.PreviousBlockHash.String()).Updates(chain)
	if err := res.Error; err != nil {
		return err
	}

	if res.RowsAffected != 1 {
		return ErrInconsistentDB
	}

	return nil
}

func (m *mainchainKeeper) processIssuing(txs []*types.Tx) error {
	for _, tx := range txs {
		for _, input := range tx.Inputs {
			switch inp := input.TypedInput.(type) {
			case *types.IssuanceInput:
				assetID := inp.AssetID()
				if _, err := m.assetStore.GetByAssetID(assetID.String()); err == nil {
					continue
				}

				m.assetStore.Add(&orm.Asset{
					AssetID:           assetID.String(),
					IssuanceProgram:   hex.EncodeToString(inp.IssuanceProgram),
					VMVersion:         inp.VMVersion,
					RawDefinitionByte: hex.EncodeToString(inp.AssetDefinition),
				})
			}
		}
	}

	return nil
}
