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
	ch                 chan VoteInfo
	overReadCH         chan struct{}
	quit               chan struct{}
	voteResults        map[string]*voteResult
	voterRewards       map[string]*voterReward
	coinBaseReward     map[string]*coinBaseReward
	period             uint64
	roundVoteBlockNums uint64
}

func NewVote(nodes []config.VoteRewardConfig, ch chan VoteInfo, overReadCH, quit chan struct{}, period uint64) *Vote {
	return &Vote{
		nodes:              nodes,
		ch:                 ch,
		overReadCH:         overReadCH,
		quit:               quit,
		voteResults:        make(map[string]*voteResult),
		voterRewards:       make(map[string]*voterReward),
		coinBaseReward:     make(map[string]*coinBaseReward),
		period:             period,
		roundVoteBlockNums: consensus.ActiveNetParams.DPOSConfig.RoundVoteBlockNums,
	}
}

func (v *Vote) Start() {
	// get coinbase reward
	if err := v.getCoinbaseReward(); err != nil {
		panic(errors.Wrap(err, "get coinbase reward"))
	}

	v.countVotes()
	v.countReward()

	// send transactions
	v.sendRewardTransaction()

}

func (v *Vote) getCoinbaseReward() error {
	if len(v.nodes) > 0 {
		tx := Transaction{
			ip: fmt.Sprintf("http://%s:%d", v.nodes[0].Host, v.nodes[0].Port),
		}
		for {
			h, err := tx.GetCurrentHeight()
			if err != nil {
				close(v.quit)
				return errors.Wrap(err, "get block height")
			}
			if h >= v.roundVoteBlockNums*v.period {
				break
			}
		}

		coinbaseTx, err := tx.GetCoinbaseTx(v.roundVoteBlockNums * v.period)
		if err != nil {
			close(v.quit)
			return err
		}
		for _, output := range coinbaseTx.Outputs {
			output, ok := output.TypedOutput.(*types.IntraChainOutput)
			if !ok {
				close(v.quit)
				return errors.New("Output type error")
			}
			address := common.GetAddressFromControlProgram(output.ControlProgram)
			for _, node := range v.nodes {
				if address == node.MiningAddress {
					reward := &coinBaseReward{
						totalReward: output.Amount,
					}
					ratioNumerator := big.NewInt(int64(node.RewardRatio))
					ratioDenominator := big.NewInt(100)
					coinBaseReward := big.NewInt(0).SetUint64(output.Amount)
					reward.voteTotalReward = coinBaseReward.Mul(coinBaseReward, ratioNumerator).Div(coinBaseReward, ratioDenominator)
					v.coinBaseReward[node.XPub] = reward
				}
			}
		}
	}
	return nil
}

func (v *Vote) countVotes() {
out:
	for {
		select {
		case voteInfo := <-v.ch:
			bigBlockNum := big.NewInt(0).SetUint64(voteInfo.VoteBlockNum)
			bigVoteNum := big.NewInt(0).SetUint64(voteInfo.VoteNum)
			bigVoteNum.Mul(bigVoteNum, bigBlockNum)

			if value, ok := v.voteResults[voteInfo.XPub]; ok {
				if vote, ok := value.votes[voteInfo.Address]; ok {
					vote.Add(vote, bigVoteNum)
				} else {
					value.votes[voteInfo.Address] = bigVoteNum
				}
			} else {
				voteResult := &voteResult{
					votes:     make(map[string]*big.Int),
					voteTotal: big.NewInt(0),
				}

				voteResult.votes[voteInfo.Address] = bigVoteNum
				v.voteResults[voteInfo.XPub] = voteResult
			}
			voteTotal := v.voteResults[voteInfo.XPub].voteTotal
			voteTotal.Add(voteTotal, bigVoteNum)
			v.voteResults[voteInfo.XPub].voteTotal = voteTotal
		case <-v.overReadCH:
			break out
		}
	}
}

func (v *Vote) countReward() {
	for xpub, votes := range v.voteResults {
		coinBaseReward, ok := v.coinBaseReward[xpub]
		if !ok {
			log.Errorf("%s doesn't have a coinbase reward \n", xpub)
			continue
		}

		for address, vote := range votes.votes {
			if value, ok := v.voterRewards[xpub]; ok {
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
					v.voterRewards[xpub] = reward
				}
			}
		}

	}
}

func (v *Vote) sendRewardTransaction() {
	for _, node := range v.nodes {
		coinbaseReward, ok := v.coinBaseReward[node.XPub]
		if !ok {
			log.Errorf("%s doesn't have a coinbase reward \n", node.XPub)
			continue
		}

		if voterRewards, ok := v.voterRewards[node.XPub]; ok {
			txID, err := v.sendReward(coinbaseReward.totalReward, node, voterRewards)
			if err != nil {
				log.WithFields(log.Fields{"error": err, "node": node}).Error("send reward transaction")
				continue
			}
			log.Info("tx_id: ", txID)
		}
	}
	close(v.quit)
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
