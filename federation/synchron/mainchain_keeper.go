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

	vaporCfg "github.com/vapor/config"
	"github.com/vapor/errors"
	"github.com/vapor/federation/common"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database"
	"github.com/vapor/federation/database/orm"
	"github.com/vapor/federation/service"
	"github.com/vapor/protocol/bc"
)

var fedProg = vaporCfg.FederationProgrom(vaporCfg.CommonConfig)

type mainchainKeeper struct {
	cfg        *config.Chain
	db         *gorm.DB
	node       *service.Node
	chainName  string
	assetCache *database.AssetCache
	txCh       chan *orm.CrossTransaction
}

func NewMainchainKeeper(db *gorm.DB, chainCfg *config.Chain, txCh chan *orm.CrossTransaction) *mainchainKeeper {
	return &mainchainKeeper{
		cfg:        chainCfg,
		db:         db,
		node:       service.NewNode(chainCfg.Upstream),
		chainName:  chainCfg.Name,
		assetCache: database.NewAssetCache(),
		txCh:       txCh,
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
	m.db.Begin()
	if err := m.processBlock(chain, block, txStatus); err != nil {
		m.db.Rollback()
		return err
	}

	return m.db.Commit().Error
}

func (m *mainchainKeeper) processBlock(chain *orm.Chain, block *types.Block, txStatus *bc.TransactionStatus) error {
	if err := m.processIssuing(block.Transactions); err != nil {
		return err
	}

	for i, tx := range block.Transactions {
		if m.isDepositTx(tx) {
			ormTx, err := m.processDepositTx(chain, block, txStatus, uint64(i), tx)
			if err != nil {
				return err
			}

			m.txCh <- ormTx
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
		if bytes.Equal(output.OutputCommitment.ControlProgram, fedProg) {
			return true
		}
	}
	return false
}

func (m *mainchainKeeper) isWithdrawalTx(tx *types.Tx) bool {
	for _, input := range tx.Inputs {
		if bytes.Equal(input.ControlProgram(), fedProg) {
			return true
		}
	}
	return false
}

func (m *mainchainKeeper) processDepositTx(chain *orm.Chain, block *types.Block, txStatus *bc.TransactionStatus, txIndex uint64, tx *types.Tx) (*orm.CrossTransaction, error) {
	blockHash := block.Hash()

	var muxID btmBc.Hash
	res0ID := tx.ResultIds[0]
	switch res := tx.Entries[*res0ID].(type) {
	case *btmBc.Output:
		muxID = *res.Source.Ref
	case *btmBc.Retirement:
		muxID = *res.Source.Ref
	default:
		return nil, ErrOutputType
	}

	rawTx, err := tx.MarshalText()
	if err != nil {
		return nil, err
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
	if err := m.db.Create(ormTx).Error; err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("create mainchain DepositTx %s", tx.ID.String()))
	}

	statusFail := txStatus.VerifyStatus[txIndex].StatusFail
	crossChainInputs, err := m.getCrossChainReqs(ormTx.ID, tx, statusFail)
	if err != nil {
		return nil, err
	}

	for _, input := range crossChainInputs {
		if err := m.db.Create(input).Error; err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("create DepositFromMainchain input: txid(%s), pos(%d)", tx.ID.String(), input.SourcePos))
		}

		ormTx.Reqs = append(ormTx.Reqs, input)
	}

	ormTx.Chain = chain
	return ormTx, nil
}

func (m *mainchainKeeper) getCrossChainReqs(crossTransactionID uint64, tx *types.Tx, statusFail bool) ([]*orm.CrossTransactionReq, error) {
	// assume inputs are from an identical owner
	script := hex.EncodeToString(tx.Inputs[0].ControlProgram())
	inputs := []*orm.CrossTransactionReq{}
	for i, rawOutput := range tx.Outputs {
		// check valid deposit
		if !bytes.Equal(rawOutput.OutputCommitment.ControlProgram, fedProg) {
			continue
		}

		if statusFail && *rawOutput.OutputCommitment.AssetAmount.AssetId != *consensus.BTMAssetID {
			continue
		}

		asset, err := m.getAsset(rawOutput.OutputCommitment.AssetAmount.AssetId.String())
		if err != nil {
			return nil, err
		}

		input := &orm.CrossTransactionReq{
			CrossTransactionID: crossTransactionID,
			SourcePos:          uint64(i),
			AssetID:            asset.ID,
			AssetAmount:        rawOutput.OutputCommitment.AssetAmount.Amount,
			Script:             script,
		}
		inputs = append(inputs, input)
	}
	return inputs, nil
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
				if _, err := m.getAsset(assetID.String()); err == nil {
					continue
				}

				asset := &orm.Asset{
					AssetID:           assetID.String(),
					IssuanceProgram:   hex.EncodeToString(inp.IssuanceProgram),
					VMVersion:         inp.VMVersion,
					RawDefinitionByte: hex.EncodeToString(inp.AssetDefinition),
				}
				if err := m.db.Create(asset).Error; err != nil {
					return err
				}

				m.assetCache.Add(asset.AssetID, asset)
			}
		}
	}

	return nil
}

func (m *mainchainKeeper) getAsset(assetID string) (*orm.Asset, error) {
	if asset := m.assetCache.Get(assetID); asset != nil {
		return asset, nil
	}

	asset := &orm.Asset{AssetID: assetID}
	if err := m.db.Where(asset).First(asset).Error; err != nil {
		return nil, errors.Wrap(err, "asset not found in memory and mysql")
	}

	m.assetCache.Add(assetID, asset)
	return asset, nil
}
