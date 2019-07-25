package consensusreward

import (
	"math/big"

	"github.com/vapor/consensus"
	"github.com/vapor/errors"
	apinode "github.com/vapor/toolbar/apinode"
	"github.com/vapor/toolbar/common"
	"github.com/vapor/toolbar/consensusreward/config"
)

var (
	errNotStandbyNode = errors.New("No Standby Node")
	errNotRewardTx    = errors.New("No reward transaction")
)

const standbyNodesRewardForConsensusCycle = 7610350076 // 400000000000000 / (365 * 24 * 60 / (500 * 1200 / 1000 / 60))

type StandbyNodeReward struct {
	cfg         *config.Config
	node        *apinode.Node
	rewards     map[string]uint64
	xpubAddress map[string]string
	startHeight uint64
	endHeight   uint64
}

func NewStandbyNodeReward(cfg *config.Config, startHeight, endHeight uint64) *StandbyNodeReward {
	s := &StandbyNodeReward{
		cfg:         cfg,
		node:        apinode.NewNode(cfg.NodeIP),
		rewards:     make(map[string]uint64),
		xpubAddress: make(map[string]string),
		startHeight: startHeight,
		endHeight:   endHeight,
	}
	for _, item := range cfg.RewardConf.Node {
		s.xpubAddress[item.XPub] = item.Address
	}
	return s
}

func (s *StandbyNodeReward) getStandbyNodeReward(height uint64) (map[string]uint64, error) {
	voteInfos, err := s.node.GetVoteByHeight(height)
	if err != nil {
		return nil, errors.Wrapf(err, "get alternative node reward")
	}
	voteInfos = common.CalcStandByNodes(voteInfos)
	if len(voteInfos) == 0 {
		return nil, errNotStandbyNode
	}
	totalVoteNum := uint64(0)
	for _, voteInfo := range voteInfos {
		totalVoteNum += voteInfo.VoteNum
	}
	total := big.NewInt(0).SetUint64(totalVoteNum)
	xpubReward := make(map[string]uint64)
	for _, voteInfo := range voteInfos {
		amount := big.NewInt(0).SetUint64(standbyNodesRewardForConsensusCycle)
		voteNum := big.NewInt(0).SetUint64(voteInfo.VoteNum)
		xpubReward[voteInfo.Vote] = amount.Mul(amount, voteNum).Div(amount, total).Uint64()
	}
	return xpubReward, nil
}

func (s *StandbyNodeReward) Settlement() error {
	if err := s.calcAllReward(); err != nil {
		return err
	}
	return s.node.BatchSendBTM(s.cfg.RewardConf.AccountID, s.cfg.RewardConf.Password, s.rewards)
}

func (s *StandbyNodeReward) calcAllReward() error {
	for height := s.startHeight; height <= s.endHeight; height += consensus.ActiveNetParams.RoundVoteBlockNums {
		xpubReward, err := s.getStandbyNodeReward(height - consensus.ActiveNetParams.RoundVoteBlockNums)
		if err == errNotStandbyNode {
			continue
		}
		if err != nil {
			return err
		}
		for xpub, amount := range xpubReward {
			if address, ok := s.xpubAddress[xpub]; ok {
				s.rewards[address] += amount
			}
		}
	}
	if len(s.rewards) == 0 {
		return errNotRewardTx
	}
	return nil
}
