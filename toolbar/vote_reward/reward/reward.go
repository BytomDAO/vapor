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

type voteResult struct {
	VoteAddress string
	VoteNum     uint64
}

type Reward struct {
	rewardCfg          *config.RewardConfig
	node               *apinode.Node
	db                 *gorm.DB
	rewards            map[string]uint64
	roundVoteBlockNums uint64
	rewardStartHeight  uint64
	rewardEndHeight    uint64
}

func NewReward(db *gorm.DB, cfg *config.Config, rewardStartHeight, rewardEndHeight uint64) *Reward {
	return &Reward{
		db:                 db,
		rewardCfg:          cfg.RewardConf,
		node:               apinode.NewNode(cfg.NodeIP),
		rewards:            make(map[string]uint64),
		roundVoteBlockNums: consensus.ActiveNetParams.DPOSConfig.RoundVoteBlockNums,
		rewardStartHeight:  rewardStartHeight,
		rewardEndHeight:    rewardEndHeight,
	}
}

func (r *Reward) getVote(height uint64) (voteResults []*voteResult, err error) {
	query := r.db.Select("vote_address, sum(vote_num) as vote_num").Model(&orm.Utxo{})
	query = query.Where("(veto_height >= ? or veto_height = 0) and vote_height <= ? and xpub = ?", height-r.roundVoteBlockNums+1, height-r.roundVoteBlockNums, r.rewardCfg.XPub)
	query = query.Group("vote_address")

	rows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var (
			address string
			voteNum uint64
		)
		rows.Scan(&address, &voteNum)
		voteResults = append(voteResults, &voteResult{
			VoteAddress: address,
			VoteNum:     voteNum,
		})
	}
	return voteResults, nil
}

func (r *Reward) Send() error {
	for height := r.rewardStartHeight + r.roundVoteBlockNums; height <= r.rewardEndHeight; height += r.roundVoteBlockNums {
		coinbaseHeight := height + 1
		coinbaseReward, err := r.getCoinbaseReward(coinbaseHeight)
		if err != nil {
			return errors.Wrapf(err, "get coinbase reward at coinbase_height: %d", coinbaseHeight)
		}

		voteResults, err := r.getVote(height)
		if err != nil {
			return err
		}

		if err := r.calcVoterRewards(voteResults, coinbaseReward); err != nil {
			return errors.Wrapf(err, "calc reaward at coinbase_height: %d", height+1)
		}
	}

	// send transactions
	return r.node.BatchSendBTM(r.rewardCfg.AccountID, r.rewardCfg.Passwd, r.rewards)
}

func (r *Reward) getCoinbaseReward(height uint64) (uint64, error) {
	block, err := r.node.GetBlockByHeight(height)
	if err != nil {
		return 0, err
	}

	miningControl, err := common.GetControlProgramFromAddress(r.rewardCfg.MiningAddress)
	if err != nil {
		return 0, err
	}

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

func (r *Reward) getTotalVoteNum(voteResults []*voteResult) (totalVoteNum uint64) {
	totalVoteNum = 0
	for _, voteResult := range voteResults {
		totalVoteNum += voteResult.VoteNum
	}
	return totalVoteNum
}

// voteNum / totalVoteNum  * (coinbaseReward * rewardRatio / 100)
func (r *Reward) calcRewardByRatio(voteNum, totalVoteNum, coinbaseReward, rewardRatio uint64) (uint64, error) {
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

func (r *Reward) calcVoterRewards(voteResults []*voteResult, coinbaseReward uint64) error {
	totalVoteNum := r.getTotalVoteNum(voteResults)
	for _, voteResult := range voteResults {
		value, err := r.calcRewardByRatio(voteResult.VoteNum, totalVoteNum, coinbaseReward, r.rewardCfg.RewardRatio)
		if err != nil {
			return err
		}

		r.rewards[voteResult.VoteAddress] += value
	}
	return nil
}
