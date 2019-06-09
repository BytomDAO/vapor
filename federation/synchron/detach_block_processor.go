package synchron

import (
	"fmt"

	btmTypes "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"

	"github.com/vapor/errors"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database/orm"
	"github.com/vapor/protocol/bc"
	vaporTypes "github.com/vapor/protocol/bc/types"
)

type detachBlockProcessor struct {
	cfg      *config.Chain
	db       *gorm.DB
	chain    *orm.Chain
	block    interface{}
	assetMap map[string]*orm.Asset
	// TransactionStatus has same marshalling rule for both bytom and vapor
	txStatus *bc.TransactionStatus
}

func (p *detachBlockProcessor) getCfg() *config.Chain {
	return p.cfg
}

func (p *detachBlockProcessor) getBlock() interface{} {
	return p.block
}

func (p *detachBlockProcessor) processIssuing(txs []*btmTypes.Tx) error {
	return nil
}

func (p *detachBlockProcessor) processDepositFromMainchain(txIndex uint64, tx *btmTypes.Tx) error {
	ormTx := &orm.CrossTransaction{ChainID: p.chain.ID, TxHash: tx.ID.String()}
	if err := p.db.Where(tx).First(tx).Error; err != nil {
		return errors.Wrap(err, "db query transaction")
	}

	if err := p.db.Delete(&orm.CrossTransactionInput{}, "MainchainTxID = ?", ormTx.ID).Error; err != nil {
		return errors.Wrap(err, fmt.Sprintf("db delete CrossTransactionInput for midechain tx %s", tx.ID.String()))
	}

	return p.db.Delete(ormTx).Error
}

func (p *detachBlockProcessor) processWithdrawalToMainchain(txIndex uint64, tx *btmTypes.Tx) error {
	return p.db.Delete(&orm.CrossTransaction{ChainID: p.chain.ID, TxHash: tx.ID.String()}).Error
}

func (p *detachBlockProcessor) processDepositToSidechain(txIndex uint64, tx *vaporTypes.Tx) error {
	return p.db.Delete(&orm.CrossTransaction{ChainID: p.chain.ID, TxHash: tx.ID.String()}).Error
}

func (p *detachBlockProcessor) processWithdrawalFromSidechain(txIndex uint64, tx *vaporTypes.Tx) error {
	ormTx := &orm.CrossTransaction{ChainID: p.chain.ID, TxHash: tx.ID.String()}
	if err := p.db.Where(tx).First(tx).Error; err != nil {
		return errors.Wrap(err, "db query transaction")
	}

	if err := p.db.Delete(&orm.CrossTransactionOutput{}, "SidechainTxID = ?", ormTx.ID).Error; err != nil {
		return errors.Wrap(err, fmt.Sprintf("db delete CrossTransactionOutput for sidechain tx %s", tx.ID.String()))
	}

	return p.db.Delete(ormTx).Error
}

func (p *detachBlockProcessor) processChainInfo() error {
	var oldBlockHashStr string

	switch {
	case p.cfg.IsMainchain:
		p.chain.BlockHash = p.block.(*btmTypes.Block).PreviousBlockHash.String()
		p.chain.BlockHeight = p.block.(*btmTypes.Block).Height - 1
		oldBlockHash := p.block.(*btmTypes.Block).Hash()
		oldBlockHashStr = oldBlockHash.String()
	default:
		p.chain.BlockHash = p.block.(*vaporTypes.Block).PreviousBlockHash.String()
		p.chain.BlockHeight = p.block.(*vaporTypes.Block).Height - 1
		oldBlockHash := p.block.(*vaporTypes.Block).Hash()
		oldBlockHashStr = oldBlockHash.String()
	}

	db := p.db.Model(p.chain).Where("block_hash = ?", oldBlockHashStr).Updates(p.chain)
	if err := db.Error; err != nil {
		return err
	}

	if db.RowsAffected != 1 {
		return ErrInconsistentDB
	}

	return nil
}
