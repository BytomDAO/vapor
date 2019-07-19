package reward

import (
	"fmt"
	"math/big"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/consensus"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/toolbar/common"
	"github.com/vapor/toolbar/reward/config"
	"github.com/vapor/toolbar/reward/database/orm"
)

type voterReward struct {
	rewards map[string]*big.Int
}

type voteResult struct {
	votes     map[string]*big.Int
	voteTotal *big.Int
}

type coinBaseReward struct {
	totalReward     uint64
	voteTotalReward *big.Int
}

type Vote struct {
	nodes              []config.VoteRewardConfig
	period             uint64
	roundVoteBlockNums uint64
	rewardStartHeight  uint64
	rewardEndHeight    uint64
}

func NewVote(nodes []config.VoteRewardConfig, rewardStartHeight, rewardEndHeight uint64) *Vote {
	return &Vote{
		nodes:              nodes,
		roundVoteBlockNums: consensus.ActiveNetParams.DPOSConfig.RoundVoteBlockNums,
		rewardStartHeight:  rewardStartHeight,
		rewardEndHeight:    rewardEndHeight,
	}
}

func (v *Vote) Start(utxos []*orm.Utxo) {
	for height := v.rewardStartHeight + v.roundVoteBlockNums; height <= v.rewardEndHeight; height += v.roundVoteBlockNums {
		// get coinbase reward
		coinbaseRewards, err := v.getCoinbaseReward(height)
		if err != nil {
			return
		}
		voteResults := v.countVotes(utxos, height)
		voterRewards := v.countReward(voteResults, coinbaseRewards)
		// send transactions
		v.sendRewardTransaction(coinbaseRewards, voterRewards)
	}

}

func (v *Vote) getCoinbaseReward(height uint64) (map[string]*coinBaseReward, error) {
	coinBaseRewards := make(map[string]*coinBaseReward)
	if len(v.nodes) > 0 {
		tx := Transaction{
			ip: fmt.Sprintf("http://%s:%d", v.nodes[0].Host, v.nodes[0].Port),
		}
		coinbaseTx, err := tx.GetCoinbaseTx(height)
		if err != nil {
			log.WithFields(log.Fields{"error": err, "coinbase_height": height}).Error("get coinbase reward")
			return coinBaseRewards, err
		}
		for _, output := range coinbaseTx.Outputs {
			output, ok := output.TypedOutput.(*types.IntraChainOutput)
			if !ok {
				log.WithFields(log.Fields{"error": err, "coinbase_height": height}).Error("Output type error")
				return coinBaseRewards, errors.New("Output type error")
			}
			address := common.GetAddressFromControlProgram(output.ControlProgram)
		out:
			for _, node := range v.nodes {
				if address == node.MiningAddress {
					reward := &coinBaseReward{
						totalReward: output.Amount,
					}
					ratioNumerator := big.NewInt(int64(node.RewardRatio))
					ratioDenominator := big.NewInt(100)
					coinBaseReward := big.NewInt(0).SetUint64(output.Amount)
					reward.voteTotalReward = coinBaseReward.Mul(coinBaseReward, ratioNumerator).Div(coinBaseReward, ratioDenominator)
					coinBaseRewards[node.XPub] = reward
					break out
				}
			}
		}
	}
	return coinBaseRewards, nil
}

func (v *Vote) countVotes(utxos []*orm.Utxo, coinBaseHeight uint64) (voteResults map[string]*voteResult) {
	voteResults = make(map[string]*voteResult)
	for _, utxo := range utxos {
		voteBlockNum := uint64(0)
		if utxo.VetoHeight < (coinBaseHeight-v.roundVoteBlockNums+1) || utxo.VoteHeight > coinBaseHeight {
			continue
		} else if utxo.VetoHeight < coinBaseHeight {
			voteBlockNum = utxo.VetoHeight - utxo.VoteHeight
		} else {
			voteBlockNum = coinBaseHeight - utxo.VoteHeight
		}

		bigBlockNum := big.NewInt(0).SetUint64(voteBlockNum)
		bigVoteNum := big.NewInt(0).SetUint64(utxo.VoteNum)
		bigVoteNum.Mul(bigVoteNum, bigBlockNum)

		if value, ok := voteResults[utxo.Xpub]; ok {
			if vote, ok := value.votes[utxo.VoterAddress]; ok {
				vote.Add(vote, bigVoteNum)
			} else {
				value.votes[utxo.VoterAddress] = bigVoteNum
			}
		} else {
			voteResult := &voteResult{
				votes:     make(map[string]*big.Int),
				voteTotal: big.NewInt(0),
			}

			voteResult.votes[utxo.VoterAddress] = bigVoteNum
			voteResults[utxo.Xpub] = voteResult
		}
		voteTotal := voteResults[utxo.Xpub].voteTotal
		voteTotal.Add(voteTotal, bigVoteNum)
		voteResults[utxo.Xpub].voteTotal = voteTotal
	}
	return
}

func (v *Vote) countReward(voteResults map[string]*voteResult, coinBaseRewards map[string]*coinBaseReward) (voterRewards map[string]*voterReward) {
	voterRewards = make(map[string]*voterReward)
	for xpub, votes := range voteResults {
		coinBaseReward, ok := coinBaseRewards[xpub]
		if !ok {
			log.Errorf("%s doesn't have a coinbase reward \n", xpub)
			continue
		}

		for address, vote := range votes.votes {
			if value, ok := voterRewards[xpub]; ok {
				mul := vote.Mul(vote, coinBaseReward.voteTotalReward)
				amount := big.NewInt(0)
				amount.Div(mul, votes.voteTotal)

				value.rewards[address] = amount
			} else {
				reward := &voterReward{
					rewards: make(map[string]*big.Int),
				}

				mul := vote.Mul(vote, coinBaseReward.voteTotalReward)
				amount := big.NewInt(0)
				amount.Div(mul, votes.voteTotal)
				if amount.Uint64() > 0 {
					reward.rewards[address] = amount
					voterRewards[xpub] = reward
				}
			}
		}

	}
	return
}

func (v *Vote) sendRewardTransaction(coinBaseRewards map[string]*coinBaseReward, voterRewards map[string]*voterReward) {
	for _, node := range v.nodes {
		coinbaseReward, ok := coinBaseRewards[node.XPub]
		if !ok {
			log.Errorf("%s doesn't have a coinbase reward \n", node.XPub)
			continue
		}

		if voterReward, ok := voterRewards[node.XPub]; ok {
			txID, err := v.sendReward(coinbaseReward.totalReward, node, voterReward)
			if err != nil {
				log.WithFields(log.Fields{"error": err, "node": node}).Error("send reward transaction")
				continue
			}
			log.Info("tx_id: ", txID)
		}
	}
}

func (v *Vote) sendReward(coinbaseReward uint64, node config.VoteRewardConfig, voterReward *voterReward) (string, error) {
	var outputAction string

	inputAction := fmt.Sprintf(inputActionFmt, coinbaseReward, node.AccountID)

	index := 0
	for address, amount := range voterReward.rewards {
		index++
		outputAction += fmt.Sprintf(outputActionFmt, amount.Uint64(), address)
		if index != len(voterReward.rewards) {
			outputAction += ","
		}
	}
	tx := Transaction{
		ip: fmt.Sprintf("http://%s:%d", node.Host, node.Port),
	}

	tmpl, err := tx.buildTx(inputAction, outputAction)
	if err != nil {
		return "", err
	}

	tmpl, err = tx.signTx(node.Passwd, *tmpl)
	if err != nil {
		return "", err
	}

	return tx.SubmitTx(tmpl.Transaction)
}
