package synchron

import (
	"encoding/hex"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	cmn "github.com/tendermint/tmlibs/common"

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

func NewChainKeeper(db *gorm.DB, cfg *config.Config, syncHeight uint64) *ChainKeeper {

	keeper := &ChainKeeper{
		cfg:        &cfg.Chain,
		db:         db,
		node:       service.NewNode(cfg.Chain.Upstream),
		syncHeight: syncHeight,
	}

	blockState := &orm.BlockState{}
	if err := db.First(blockState).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			blockStr, _, err := keeper.node.GetBlockByHeight(0)
			if err != nil {
				cmn.Exit(cmn.Fmt("Failed to get genenis block:[%s]", err.Error()))
			}
			block := &types.Block{}
			if err := block.UnmarshalText([]byte(blockStr)); err != nil {
				cmn.Exit(cmn.Fmt("unmarshal block:[%s]", err.Error()))
			}
			if err := keeper.insertBlockState(db, block); err != nil {
				cmn.Exit(cmn.Fmt("Failed to insert blockState:[%s]", err.Error()))
			}
		} else {
			cmn.Exit(cmn.Fmt("Failed to get blockState:[%s]", err.Error()))
		}
	}

	return keeper
}

func (c *ChainKeeper) Start() {

	for {
		blockState := &orm.BlockState{}
		if err := c.db.First(blockState).Error; err != nil {
			log.WithField("error", err).Errorln("query blockState fail on process block")
			break
		}

		if blockState.Height >= c.syncHeight {
			break
		}

		isUpdate, err := c.syncBlock(blockState)
		if err != nil {
			log.WithField("error", err).Errorln("blockKeeper fail on process block")
			break
		}

		if !isUpdate {
			break
		}
	}
}

func (c *ChainKeeper) syncBlock(blockState *orm.BlockState) (bool, error) {
	height, err := c.node.GetBlockCount()
	if err != nil {
		return false, err
	}

	if height == blockState.Height {
		return false, nil
	}

	nextBlockStr, txStatus, err := c.node.GetBlockByHeight(blockState.Height + 1)
	if err != nil {
		return false, err
	}

	nextBlock := &types.Block{}
	if err := nextBlock.UnmarshalText([]byte(nextBlockStr)); err != nil {
		return false, errors.New("Unmarshal nextBlock")
	}

	// Normal case, the previous hash of next block equals to the hash of current block,
	// just sync to database directly.
	if nextBlock.PreviousBlockHash.String() == blockState.BlockHash {
		return true, c.AttachBlock(nextBlock, txStatus)
	}

	log.WithField("block height", blockState.Height).Debug("the prev hash of remote is not equals the hash of current best block, must rollback")
	currentBlockStr, txStatus, err := c.node.GetBlockByHash(blockState.BlockHash)
	if err != nil {
		return false, err
	}

	currentBlock := &types.Block{}
	if err := nextBlock.UnmarshalText([]byte(currentBlockStr)); err != nil {
		return false, errors.New("Unmarshal currentBlock")
	}

	return true, c.DetachBlock(currentBlock, txStatus)
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
		return err
	}

	return ormDB.Commit().Error
}

func (c *ChainKeeper) insertBlockState(db *gorm.DB, block *types.Block) error {
	blockHash := block.Hash()
	blockState := &orm.BlockState{
		Height:    block.Height,
		BlockHash: blockHash.String(),
	}
	if err := db.Save(blockState).Error; err != nil {
		return err
	}
	return nil
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
		db.Rollback()
		return err
	}

	if u.RowsAffected != 1 {
		db.Rollback()
		return ErrInconsistentDB
	}
	return nil
}
