package synchron

import (
	// "database/sql"
	// "encoding/hex"
	// "encoding/json"
	"fmt"
	// "math/big"
	// "sort"

	// "github.com/bytom/consensus"
	// TODO:
	btmBc "github.com/bytom/protocol/bc"
	btmTypes "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"

	"github.com/vapor/errors"
	"github.com/vapor/federation/common"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database/orm"
	vaporTypes "github.com/vapor/protocol/bc/types"
)

type attachBlockProcessor struct {
	cfg   *config.Chain
	db    *gorm.DB
	chain *orm.Chain
	block interface{}
	// txStatus *btmBc.TransactionStatus
}

func (p *attachBlockProcessor) getCfg() *config.Chain {
	return p.cfg
}

func (p *attachBlockProcessor) getBlock() interface{} {
	return p.block
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
		p.db.Rollback()
		return errors.Wrap(err, fmt.Sprintf("create DepositFromMainchain tx %s", tx.ID.String()))
	}

	for i, input := range getCrossChainInputs(ormTx.ID, tx) {
		if err := p.db.Create(input).Error; err != nil {
			p.db.Rollback()
			return errors.Wrap(err, fmt.Sprintf("create DepositFromMainchain input: txid(%s), pos(%d)", tx.ID.String(), i))
		}
	}

	return nil
}

func (p *attachBlockProcessor) processIssuing(db *gorm.DB, txs []*btmTypes.Tx) error {
	return addIssueAssets(db, txs)
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
