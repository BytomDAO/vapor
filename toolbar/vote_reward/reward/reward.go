package reward

import (
	"bytes"

	"github.com/jinzhu/gorm"

	"github.com/vapor/consensus"
	"github.com/vapor/errors"
	"github.com/vapor/math/checked"
	"github.com/vapor/protocol/bc/types"
	apinode "github.com/vapor/toolbar/api_node"
	"github.com/vapor/toolbar/common"
	"github.com/vapor/toolbar/vote_reward/config"
	"github.com/vapor/toolbar/vote_reward/database/orm"
)

type voterReward struct {
	rewards map[string]uint64
}

type voteResult struct {
	VoteAddress string
	VoteNum     uint64
}

type Vote struct {
	rewardCfg *config.RewardConfig
	node      *apinode.Node
	db        *gorm.DB
	reward    *voterReward
	/*
		reward             *voterReward
		coinBaseRewards    map[uint64]*coinBaseReward
	*/
	roundVoteBlockNums uint64
	rewardStartHeight  uint64
	rewardEndHeight    uint64
}

func NewVote(db *gorm.DB, cfg *config.Config, rewardStartHeight, rewardEndHeight uint64) *Vote {
	return &Vote{
		db:                 db,
		rewardCfg:          cfg.RewardConf,
		node:               apinode.NewNode(cfg.NodeIP),
		reward:             &voterReward{rewards: make(map[string]uint64)},
		roundVoteBlockNums: consensus.ActiveNetParams.DPOSConfig.RoundVoteBlockNums,
		rewardStartHeight:  rewardStartHeight,
		rewardEndHeight:    rewardEndHeight,
	}
}

func (v *Vote) getVote(height uint64) (voteResults []*voteResult, err error) {
	query := v.db.Model(&orm.Utxo{}).Select("vote_address, sum(vote_num) as vote_num")
	query.Where("(veto_height >= ? or veto_height = 0) and vote_height <= ? and xpub = ?", height-v.roundVoteBlockNums+1, height-v.roundVoteBlockNums, v.rewardCfg.XPub)
	query.Group("vote_address")

	voteResults = []*voteResult{}
	if err = query.Scan(&voteResults).Error; err != nil {
		return nil, err
	}

	return voteResults, nil
}

func (v *Vote) Start() error {
	for height := v.rewardStartHeight + v.roundVoteBlockNums; height <= v.rewardEndHeight; height += v.roundVoteBlockNums {
		coinbaseHeight := height + 1
		coinbaseReward, err := v.getCoinbaseReward(coinbaseHeight)
		if err != nil {
			return errors.Wrapf(err, "get coinbase reward at coinbase_height: %d", coinbaseHeight)
		}

		voteResults, err := v.getVote(height)
		if err != nil {
			return errors.Wrapf(err, "get vote from db at coinbase_height: %d", height)
		}

		if err := v.calcVoterRewards(voteResults, coinbaseReward); err != nil {
			return errors.Wrapf(err, "calc reaward at coinbase_height: %d", height+1)
		}
	}

	// send transactions
	return v.node.BatchSendBTM(v.rewardCfg.AccountID, v.rewardCfg.Passwd, v.reward.rewards)
}

func (v *Vote) getCoinbaseReward(height uint64) (uint64, error) {
	block, err := v.node.GetBlockByHeight(height)
	if err != nil {
		return 0, err
	}

	miningControl := common.GetControlProgramFromAddress(v.rewardCfg.MiningAddress)
	for _, output := range block.Transactions[0].Outputs {
		output, ok := output.TypedOutput.(*types.IntraChainOutput)
		if !ok {
			return 0, errors.New("Output type error")
		}

		if output.Amount == 0 {
			continue
		}

		if bytes.Equal(miningControl, output.ControlProgram) {
			return output.Amount, nil
		}
	}
	return 0, errors.New("No reward found")
}

func (v *Vote) getTotalVoteNum(voteResults []*voteResult) (totalVoteNum uint64) {
	totalVoteNum = 0
	for _, voteResult := range voteResults {
		totalVoteNum += voteResult.VoteNum
	}
	return totalVoteNum
}

// voteNum / totalVoteNum  * (coinbaseReward * rewardRatio / 100)
func (v *Vote) calcRewardByRatio(voteNum, totalVoteNum, coinbaseReward, rewardRatio uint64) (uint64, error) {
	reward := uint64(0)
	mul, ok := checked.MulUint64(coinbaseReward, rewardRatio)
	if !ok {
		return 0, checked.ErrOverflow
	}

	reward, ok = checked.DivUint64(mul, 100)
	if !ok {
		return 0, checked.ErrOverflow
	}

	mul, ok = checked.MulUint64(voteNum, reward)
	if !ok {
		return 0, checked.ErrOverflow
	}

	reward, ok = checked.DivUint64(mul, totalVoteNum)
	if !ok {
		return 0, checked.ErrOverflow
	}

	return reward, nil
}

func (v *Vote) calcVoterRewards(voteResults []*voteResult, coinbaseReward uint64) error {
	totalVoteNum := v.getTotalVoteNum(voteResults)
	rewardRatio := uint64(v.rewardCfg.RewardRatio)
	for _, voteResult := range voteResults {
		value, err := v.calcRewardByRatio(voteResult.VoteNum, totalVoteNum, coinbaseReward, rewardRatio)
		if err != nil {
			return err
		}

		v.reward.rewards[voteResult.VoteAddress] += value
	}
	return nil
}
