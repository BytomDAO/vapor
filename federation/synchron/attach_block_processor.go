package synchron

import (
	"encoding/hex"
	// "encoding/json"
	"fmt"

	// "github.com/bytom/consensus"
	btmBc "github.com/bytom/protocol/bc"
	btmTypes "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"

	"github.com/vapor/errors"
	"github.com/vapor/federation/common"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database/orm"
	vaporBc "github.com/vapor/protocol/bc"
	vaporTypes "github.com/vapor/protocol/bc/types"
)

type attachBlockProcessor struct {
	cfg      *config.Chain
	db       *gorm.DB
	chain    *orm.Chain
	block    interface{}
	assetMap map[string]*orm.Asset
	// txStatus *btmBc.TransactionStatus
}

func (p *attachBlockProcessor) getCfg() *config.Chain {
	return p.cfg
}

func (p *attachBlockProcessor) getBlock() interface{} {
	return p.block
}

func (p *attachBlockProcessor) processIssuing(txs []*btmTypes.Tx) error {
	var assets []*orm.Asset

	for _, tx := range txs {
		for _, input := range tx.Inputs {
			switch inp := input.TypedInput.(type) {
			case *btmTypes.IssuanceInput:
				assetID := inp.AssetID()
				if _, ok := p.assetMap[assetID.String()]; ok {
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
		if err := p.db.Create(asset).Error; err != nil {
			return err
		}

		p.assetMap[asset.AssetID] = asset
	}

	return nil
}

func (p *attachBlockProcessor) processDepositFromMainchain(txIndex uint64, tx *btmTypes.Tx) error {
	blockHash := p.getBlock().(*btmTypes.Block).Hash()

	var muxID btmBc.Hash
	resOutID := tx.ResultIds[0]
	resOut, ok := tx.Entries[*resOutID].(*btmBc.Output)
	if ok {
		muxID = *resOut.Source.Ref
	} else {
		return errors.New("fail to get mux id")
	}

	rawTx, err := tx.MarshalText()
	if err != nil {
		return err
	}

	ormTx := &orm.CrossTransaction{
		ChainID:        p.chain.ID,
		Direction:      common.DepositDirection,
		BlockHeight:    p.getBlock().(*btmTypes.Block).Height,
		BlockHash:      blockHash.String(),
		TxIndex:        txIndex,
		MuxID:          muxID.String(),
		TxHash:         tx.ID.String(),
		RawTransaction: string(rawTx),
		Status:         common.CrossTxCompletedStatus,
	}
	if err := p.db.Create(ormTx).Error; err != nil {
		return errors.Wrap(err, fmt.Sprintf("create DepositFromMainchain tx %s", tx.ID.String()))
	}

	crossChainInputs, err := getCrossChainInputs(ormTx.ID, tx, p.assetMap)
	if err != nil {
		return err
	}

	for _, input := range crossChainInputs {
		if err := p.db.Create(input).Error; err != nil {
			return errors.Wrap(err, fmt.Sprintf("create DepositFromMainchain input: txid(%s), pos(%d)", tx.ID.String(), input.SourcePos))
		}
	}

	return nil
}

func (p *attachBlockProcessor) processWithdrawalToMainchain(txIndex uint64, tx *btmTypes.Tx) error {
	blockHash := p.getBlock().(*btmTypes.Block).Hash()

	var muxID btmBc.Hash
	resOutID := tx.ResultIds[0]
	resOut, ok := tx.Entries[*resOutID].(*btmBc.Output)
	if ok {
		muxID = *resOut.Source.Ref
	} else {
		return errors.New("fail to get mux id")
	}

	rawTx, err := tx.MarshalText()
	if err != nil {
		return err
	}

	ormTx := &orm.CrossTransaction{
		ChainID:        p.chain.ID,
		Direction:      common.WithdrawalDirection,
		BlockHeight:    p.getBlock().(*btmTypes.Block).Height,
		BlockHash:      blockHash.String(),
		TxIndex:        txIndex,
		MuxID:          muxID.String(),
		TxHash:         tx.ID.String(),
		RawTransaction: string(rawTx),
		Status:         common.CrossTxCompletedStatus,
	}
	if err := p.db.Create(ormTx).Error; err != nil {
		return errors.Wrap(err, fmt.Sprintf("create DepositFromMainchain tx %s", tx.ID.String()))
	}

	return nil
}

func (p *attachBlockProcessor) processDepositToSidechain(txIndex uint64, tx *vaporTypes.Tx) error {
	blockHash := p.getBlock().(*vaporTypes.Block).Hash()

	var muxID vaporBc.Hash
	resOutID := tx.ResultIds[0]
	resOut, ok := tx.Entries[*resOutID].(*vaporBc.IntraChainOutput)
	if ok {
		muxID = *resOut.Source.Ref
	} else {
		return errors.New("fail to get mux id")
	}

	rawTx, err := tx.MarshalText()
	if err != nil {
		return err
	}

	ormTx := &orm.CrossTransaction{
		ChainID:        p.chain.ID,
		Direction:      common.DepositDirection,
		BlockHeight:    p.getBlock().(*vaporTypes.Block).Height,
		BlockHash:      blockHash.String(),
		TxIndex:        txIndex,
		MuxID:          muxID.String(),
		TxHash:         tx.ID.String(),
		RawTransaction: string(rawTx),
		Status:         common.CrossTxCompletedStatus,
	}
	if err := p.db.Create(ormTx).Error; err != nil {
		return errors.Wrap(err, fmt.Sprintf("create DepositToSidechain tx %s", tx.ID.String()))
	}

	return nil
}

func (p *attachBlockProcessor) processWithdrawalFromSidechain(txIndex uint64, tx *vaporTypes.Tx) error {
	blockHash := p.getBlock().(*vaporTypes.Block).Hash()

	var muxID vaporBc.Hash
	resOutID := tx.ResultIds[0]
	resOut, ok := tx.Entries[*resOutID].(*vaporBc.CrossChainOutput)
	if ok {
		muxID = *resOut.Source.Ref
	} else {
		return errors.New("fail to get mux id")
	}

	rawTx, err := tx.MarshalText()
	if err != nil {
		return err
	}

	ormTx := &orm.CrossTransaction{
		ChainID:        p.chain.ID,
		Direction:      common.WithdrawalDirection,
		BlockHeight:    p.getBlock().(*vaporTypes.Block).Height,
		BlockHash:      blockHash.String(),
		TxIndex:        txIndex,
		MuxID:          muxID.String(),
		TxHash:         tx.ID.String(),
		RawTransaction: string(rawTx),
		Status:         common.CrossTxCompletedStatus,
	}
	if err := p.db.Create(ormTx).Error; err != nil {
		return errors.Wrap(err, fmt.Sprintf("create WithdrawalFromSidechain tx %s", tx.ID.String()))
	}

	for i, output := range getCrossChainOutputs(ormTx.ID, tx) {
		if err := p.db.Create(output).Error; err != nil {
			return errors.Wrap(err, fmt.Sprintf("create WithdrawalFromSidechain output: txid(%s), pos(%d)", tx.ID.String(), i))
		}
	}

	return nil
}

func (p *attachBlockProcessor) processChainInfo() error {
	var previousBlockHashStr string

	switch {
	case p.cfg.IsMainchain:
		blockHash := p.block.(*btmTypes.Block).Hash()
		p.chain.BlockHash = blockHash.String()
		p.chain.BlockHeight = p.block.(*btmTypes.Block).Height
		previousBlockHashStr = p.block.(*btmTypes.Block).PreviousBlockHash.String()
	default:
		blockHash := p.block.(*vaporTypes.Block).Hash()
		p.chain.BlockHash = blockHash.String()
		p.chain.BlockHeight = p.block.(*vaporTypes.Block).Height
		previousBlockHashStr = p.block.(*vaporTypes.Block).PreviousBlockHash.String()
	}

	db := p.db.Model(p.chain).Where("block_hash = ?", previousBlockHashStr).Updates(p.chain)
	if err := db.Error; err != nil {
		return err
	}

	if db.RowsAffected != 1 {
		return ErrInconsistentDB
	}

	return nil
}

/*

func (p *attachBlockProcessor) getTxStatus() *bc.TransactionStatus {
	return p.txStatus
}

*/
