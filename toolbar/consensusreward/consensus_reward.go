package consensusreward

import (
	"math/big"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/toolbar/apinode"
	"github.com/bytom/vapor/toolbar/common"
	"github.com/bytom/vapor/toolbar/consensusreward/config"
)

const standbyNodesRewardForConsensusCycle = 7610350076 // 400000000000000 / (365 * 24 * 60 / (500 * 1200 / 1000 / 60))

type StandbyNodeReward struct {
	cfg         *config.Config
	node        *apinode.Node
	xpubRewards map[string]uint64
	startHeight uint64
	endHeight   uint64
}

func NewStandbyNodeReward(cfg *config.Config, startHeight, endHeight uint64) *StandbyNodeReward {
	return &StandbyNodeReward{
		cfg:         cfg,
		node:        apinode.NewNode(cfg.NodeIP),
		xpubRewards: make(map[string]uint64),
		startHeight: startHeight,
		endHeight:   endHeight,
	}
}

func (s *StandbyNodeReward) getStandbyNodeReward(height uint64) error {
	voteInfos, err := s.node.GetVoteByHeight(height)
	if err != nil {
		return errors.Wrapf(err, "get alternative node reward")
	}

	voteInfos = common.CalcStandByNodes(voteInfos)
	totalVoteNum := uint64(0)
	for _, voteInfo := range voteInfos {
		totalVoteNum += voteInfo.VoteNum
	}

	total := big.NewInt(0).SetUint64(totalVoteNum)
	for _, voteInfo := range voteInfos {
		amount := big.NewInt(0).SetUint64(standbyNodesRewardForConsensusCycle)
		voteNum := big.NewInt(0).SetUint64(voteInfo.VoteNum)
		s.xpubRewards[voteInfo.Vote] += amount.Mul(amount, voteNum).Div(amount, total).Uint64()
	}
	return nil
}

func (s *StandbyNodeReward) Settlement() error {
	for height := s.startHeight + consensus.ActiveNetParams.RoundVoteBlockNums; height <= s.endHeight; height += consensus.ActiveNetParams.RoundVoteBlockNums {
		if err := s.getStandbyNodeReward(height - consensus.ActiveNetParams.RoundVoteBlockNums); err != nil {
			return err
		}
	}

	rewards := map[string]uint64{}
	for _, item := range s.cfg.RewardConf.Node {
		if reward, ok := s.xpubRewards[item.XPub]; ok {
			rewards[item.Address] = reward
		}
	}

	if len(rewards) == 0 {
		return nil
	}

	txID, err := s.node.BatchSendBTM(s.cfg.RewardConf.AccountID, s.cfg.RewardConf.Password, rewards, []byte{})
	if err == nil {
		log.WithFields(log.Fields{
			"tx_hash":      txID,
			"start_height": s.startHeight,
			"end_height":   s.endHeight,
		}).Info("success on submit consensus reward transaction")
	}
	return err
}
