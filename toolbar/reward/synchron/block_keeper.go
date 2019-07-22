package synchron

import (
	"encoding/hex"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/toolbar/common"
	"github.com/vapor/toolbar/common/service"
	"github.com/vapor/toolbar/reward/config"
	"github.com/vapor/toolbar/reward/database/orm"
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
	if err := db.First(blockState).Error; err == gorm.ErrRecordNotFound {
		blockStr, _, err := keeper.node.GetBlockByHeight(0)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get genenis block")
		}
		block := &types.Block{}
		if err := block.UnmarshalText([]byte(blockStr)); err != nil {
			return nil, errors.Wrap(err, "unmarshal block")
		}
		if err := keeper.initBlockState(db, block); err != nil {
			return nil, errors.Wrap(err, "Failed to insert blockState")
		}
	} else if err != nil {
		return nil, errors.Wrap(err, "Failed to get blockState")
	}

	return keeper, nil
}

func (c *ChainKeeper) Start() error {
	for {
		blockState := &orm.BlockState{}
		if c.db.First(blockState).RecordNotFound() {
			return errors.New("The query blockState record is empty empty on process block")
		}

		if blockState.Height >= c.syncHeight {
			break
		}

		if err := c.syncBlock(blockState); err != nil {
			return err
		}
	}
	return nil
}

func (c *ChainKeeper) syncBlock(blockState *orm.BlockState) error {
	height, err := c.node.GetBlockCount()
	if err != nil {
		return err
	}

	if height == blockState.Height {
		return nil
	}

	nextBlockStr, txStatus, err := c.node.GetBlockByHeight(blockState.Height + 1)
	if err != nil {
		return err
	}

	nextBlock := &types.Block{}
	if err := nextBlock.UnmarshalText([]byte(nextBlockStr)); err != nil {
		return errors.New("Unmarshal nextBlock")
	}

	// Normal case, the previous hash of next block equals to the hash of current block,
	// just sync to database directly.
	if nextBlock.PreviousBlockHash.String() == blockState.BlockHash {
		return c.AttachBlock(nextBlock, txStatus)
	}

	log.WithField("block height", blockState.Height).Debug("the prev hash of remote is not equals the hash of current best block, must rollback")
	currentBlockStr, txStatus, err := c.node.GetBlockByHash(blockState.BlockHash)
	if err != nil {
		return err
	}

	currentBlock := &types.Block{}
	if err := nextBlock.UnmarshalText([]byte(currentBlockStr)); err != nil {
		return errors.New("Unmarshal currentBlock")
	}

	return c.DetachBlock(currentBlock, txStatus)
}

func (c *ChainKeeper) AttachBlock(block *types.Block, txStatus *bc.TransactionStatus) error {
	ormDB := c.db.Begin()
	for pos, tx := range block.Transactions {
		statusFail, err := txStatus.GetStatus(pos)
		if err != nil {
			return err
		}

		if statusFail {
			log.WithFields(log.Fields{"block height": block.Height, "statusFail": statusFail}).Debug("AttachBlock")
			continue
		}

		for _, input := range tx.Inputs {
			vetoInput, ok := input.TypedInput.(*types.VetoInput)
			if !ok {
				continue
			}

			outputID, err := input.SpentOutputID()
			if err != nil {
				return err
			}
			utxo := &orm.Utxo{
				VoterAddress: common.GetAddressFromControlProgram(vetoInput.ControlProgram),
				OutputID:     outputID.String(),
			}
			// update data
			db := ormDB.Model(&orm.Utxo{}).Where(utxo).Update("veto_height", block.Height)
			if err := db.Error; err != nil {
				ormDB.Rollback()
				return err
			}

			if db.RowsAffected != 1 {
				ormDB.Rollback()
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
				ormDB.Rollback()
				return err
			}
		}
	}

	if err := c.updateBlockState(ormDB, block); err != nil {
		ormDB.Rollback()
		return err
	}

	return ormDB.Commit().Error
}

func (c *ChainKeeper) DetachBlock(block *types.Block, txStatus *bc.TransactionStatus) error {
	ormDB := c.db.Begin()

	utxo := &orm.Utxo{
		VoteHeight: block.Height,
	}
	// insert data
	if err := ormDB.Where(utxo).Delete(&orm.Utxo{}).Error; err != nil {
		ormDB.Rollback()
		return err
	}

	utxo = &orm.Utxo{
		VetoHeight: block.Height,
	}

	// update data
	if err := ormDB.Where(utxo).Update("veto_height", 0).Error; err != nil {
		ormDB.Rollback()
		return err
	}

	preBlockStr, _, err := c.node.GetBlockByHeight(block.Height + 1)
	if err != nil {
		return err
	}

	preBlock := &types.Block{}
	if err := preBlock.UnmarshalText([]byte(preBlockStr)); err != nil {
		return errors.New("Unmarshal preBlock")
	}

	if err := c.updateBlockState(ormDB, preBlock); err != nil {
		ormDB.Rollback()
		return err
	}

	return ormDB.Commit().Error
}

func (c *ChainKeeper) initBlockState(db *gorm.DB, block *types.Block) error {
	blockHash := block.Hash()
	blockState := &orm.BlockState{
		Height:    block.Height,
		BlockHash: blockHash.String(),
	}

	return db.Save(blockState).Error
}

func (c *ChainKeeper) updateBlockState(db *gorm.DB, block *types.Block) error {
	// update blockState
	blockHash := block.Hash()
	blockState := &orm.BlockState{
		Height:    block.Height,
		BlockHash: blockHash.String(),
	}

	u := db.Model(&orm.BlockState{}).Updates(blockState)

	if err := u.Error; err != nil {
		return err
	}

	if u.RowsAffected != 1 {
		return ErrInconsistentDB
	}
	return nil
}
