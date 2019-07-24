package settlementvotereward

import (
	"bytes"
	"math/big"

	"github.com/jinzhu/gorm"

	"github.com/vapor/consensus"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/toolbar/apinode"
	"github.com/vapor/toolbar/common"
	"github.com/vapor/toolbar/vote_reward/config"
)

type voteResult struct {
	VoteAddress string
	VoteNum     uint64
}

type SettlementReward struct {
	rewardCfg   *config.RewardConfig
	node        *apinode.Node
	db          *gorm.DB
	rewards     map[string]uint64
	startHeight uint64
	endHeight   uint64
}

func NewSettlementReward(db *gorm.DB, cfg *config.Config, startHeight, endHeight uint64) *SettlementReward {
	return &SettlementReward{
		db:          db,
		rewardCfg:   cfg.RewardConf,
		node:        apinode.NewNode(cfg.NodeIP),
		rewards:     make(map[string]uint64),
		startHeight: startHeight,
		endHeight:   endHeight,
	}
}

func (s *SettlementReward) getVoteResultFromDB(height uint64) (voteResults []*voteResult, err error) {
	query := s.db.Table("utxos").Select("vote_address, sum(vote_num) as vote_num")
	query = query.Where("(veto_height >= ? or veto_height = 0) and vote_height <= ? and xpub = ?", height-consensus.ActiveNetParams.RoundVoteBlockNums+1, height-consensus.ActiveNetParams.RoundVoteBlockNums, s.rewardCfg.XPub)
	query = query.Group("vote_address")
	if err := query.Scan(&voteResults).Error; err != nil {
		return nil, err
	}

	return voteResults, nil
}

func (s *SettlementReward) Settlement() error {
	for height := s.startHeight + consensus.ActiveNetParams.RoundVoteBlockNums; height <= s.endHeight; height += consensus.ActiveNetParams.RoundVoteBlockNums {
		coinbaseHeight := height + 1
		totalReward, err := s.getCoinbaseReward(coinbaseHeight)
		if err != nil {
			return errors.Wrapf(err, "get total reward at coinbase_height: %d", coinbaseHeight)
		}

		voteResults, err := s.getVoteResultFromDB(height)
		if err != nil {
			return err
		}

		s.calcVoterRewards(voteResults, totalReward)
	}

	// send transactions
	return s.node.BatchSendBTM(s.rewardCfg.AccountID, s.rewardCfg.Password, s.rewards)
}

func (s *SettlementReward) getCoinbaseReward(height uint64) (uint64, error) {
	block, err := s.node.GetBlockByHeight(height)
	if err != nil {
		return 0, err
	}

	miningControl, err := common.GetControlProgramFromAddress(s.rewardCfg.MiningAddress)
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
			amount := big.NewInt(0).SetUint64(output.Amount)
			rewardRatio := big.NewInt(0).SetUint64(s.rewardCfg.RewardRatio)
			amount.Mul(amount, rewardRatio).Div(amount, big.NewInt(100))

			return amount.Uint64(), nil
		}
	}
	return 0, errors.New("No reward found")
}

func (s *SettlementReward) calcVoterRewards(voteResults []*voteResult, totalReward uint64) {
	totalVoteNum := uint64(0)
	for _, voteResult := range voteResults {
		totalVoteNum += voteResult.VoteNum
	}

	for _, voteResult := range voteResults {
		// voteNum / totalVoteNum  * totalReward
		voteNum := big.NewInt(0).SetUint64(voteResult.VoteNum)
		total := big.NewInt(0).SetUint64(totalVoteNum)
		reward := big.NewInt(0).SetUint64(totalReward)

		amount := voteNum.Mul(voteNum, reward).Div(voteNum, total).Uint64()

		if amount != 0 {
			s.rewards[voteResult.VoteAddress] += amount
		}
	}
}
