package synchron

import (
	"encoding/hex"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/toolbar/common"
	"github.com/vapor/toolbar/reward/config"
	"github.com/vapor/toolbar/reward/database/orm"
	"github.com/vapor/toolbar/reward/service"
)

type ChainKeeper struct {
	cfg        *config.Chain
	db         *gorm.DB
	node       *service.Node
	syncHeight uint64
}

func NewChainKeeper(db *gorm.DB, cfg *config.Config, syncHeight uint64) (*ChainKeeper, error) {
	keeper := &ChainKeeper{
		cfg:        &cfg.Chain,
		db:         db,
		node:       service.NewNode(cfg.Chain.Upstream),
		syncHeight: syncHeight,
	}

	blockState := &orm.BlockState{}
	err := db.First(blockState).Error
	if err == nil {
		return keeper, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, errors.Wrap(err, "Failed to get blockState")
	}

	block, err := keeper.node.GetBlockByHeight(0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get genenis block")
	}

	if err := keeper.initBlockState(db, block); err != nil {
		return nil, errors.Wrap(err, "Failed to insert blockState")
	}

	return keeper, nil
}

func (c *ChainKeeper) SyncBlock() error {
	for {
		blockState := &orm.BlockState{}
		if err := c.db.First(blockState).Error; err != nil {
			return errors.Wrap(err, "The query blockState record is empty empty on process block")
		}

		if blockState.Height >= c.syncHeight {
			break
		}
		ormDB := c.db.Begin()
		if err := c.syncBlock(ormDB, blockState); err != nil {
			ormDB.Rollback()
			return err
		}

		if err := ormDB.Commit().Error; err != nil {
			return err
		}
	}
	return nil
}

func (c *ChainKeeper) syncBlock(ormDB *gorm.DB, blockState *orm.BlockState) error {
	height, err := c.node.GetBlockCount()
	if err != nil {
		return err
	}

	if height == blockState.Height {
		return nil
	}

	nextBlock, err := c.node.GetBlockByHeight(blockState.Height + 1)
	if err != nil {
		return err
	}

	// Normal case, the previous hash of next block equals to the hash of current block,
	// just sync to database directly.
	if nextBlock.PreviousBlockHash.String() == blockState.BlockHash {
		return c.AttachBlock(ormDB, nextBlock)
	}

	log.WithField("block height", blockState.Height).Debug("the prev hash of remote is not equals the hash of current best block, must rollback")
	currentBlock, err := c.node.GetBlockByHash(blockState.BlockHash)
	if err != nil {
		return err
	}

	return c.DetachBlock(ormDB, currentBlock)
}

func (c *ChainKeeper) AttachBlock(ormDB *gorm.DB, block *types.Block) error {
	for _, tx := range block.Transactions {
		for _, input := range tx.Inputs {
			if _, ok := input.TypedInput.(*types.VetoInput); !ok {
				continue
			}

			outputID, err := input.SpentOutputID()
			if err != nil {
				return err
			}
			utxo := &orm.Utxo{
				OutputID: outputID.String(),
			}
			// update data
			db := ormDB.Model(&orm.Utxo{}).Where(utxo).Update("veto_height", block.Height)
			if err := db.Error; err != nil {
				return err
			}

			if db.RowsAffected != 1 {
				return ErrInconsistentDB
			}

		}

		for index, output := range tx.Outputs {
			voteOutput, ok := output.TypedOutput.(*types.VoteOutput)
			if !ok {
				continue
			}
			pubkey := hex.EncodeToString(voteOutput.Vote)
			outputID := tx.OutputID(index)
			utxo := &orm.Utxo{
				Xpub:         pubkey,
				VoterAddress: common.GetAddressFromControlProgram(voteOutput.ControlProgram),
				VoteHeight:   block.Height,
				VoteNum:      voteOutput.Amount,
				VetoHeight:   0,
				OutputID:     outputID.String(),
			}
			// insert data
			if err := ormDB.Save(utxo).Error; err != nil {
				return err
			}
		}
	}

	blockHash := block.Hash()
	blockState := &orm.BlockState{
		Height:    block.Height,
		BlockHash: blockHash.String(),
	}

	return c.updateBlockState(ormDB, blockState)
}

func (c *ChainKeeper) DetachBlock(ormDB *gorm.DB, block *types.Block) error {
	utxo := &orm.Utxo{
		VoteHeight: block.Height,
	}
	// insert data
	if err := ormDB.Where(utxo).Delete(&orm.Utxo{}).Error; err != nil {
		return err
	}

	utxo = &orm.Utxo{
		VetoHeight: block.Height,
	}

	// update data
	if err := ormDB.Where(utxo).Update("veto_height", 0).Error; err != nil {
		return err
	}

	blockState := &orm.BlockState{
		Height:    block.Height - 1,
		BlockHash: block.PreviousBlockHash.String(),
	}

	return c.updateBlockState(ormDB, blockState)
}

func (c *ChainKeeper) initBlockState(db *gorm.DB, block *types.Block) error {
	blockHash := block.Hash()
	blockState := &orm.BlockState{
		Height:    block.Height,
		BlockHash: blockHash.String(),
	}

	return db.Save(blockState).Error
}

func (c *ChainKeeper) updateBlockState(db *gorm.DB, blockState *orm.BlockState) error {
	// update blockState
	u := db.Model(&orm.BlockState{}).Updates(blockState)
	if err := u.Error; err != nil {
		return err
	}

	if u.RowsAffected != 1 {
		return ErrInconsistentDB
	}
	return nil
}
