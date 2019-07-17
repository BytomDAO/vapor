package reward

import (
	"fmt"
	"math/big"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/toolbar/common"
	"github.com/vapor/toolbar/reward/config"
)

type VoterReward struct {
	rewards map[string]*big.Int
}

type VoteResult struct {
	Votes     map[string]*big.Int
	VoteTotal *big.Int
}

type Vote struct {
	nodes          []config.VoteRewardConfig
	ch             chan VoteInfo
	overReadCH     chan struct{}
	voteResults    map[string]*VoteResult
	voterRewards   map[string]*VoterReward
	coinBaseReward map[string]uint64
	period         uint64
}

func NewVote(nodes []config.VoteRewardConfig, ch chan VoteInfo, overReadCH chan struct{}, period uint64) *Vote {
	return &Vote{
		nodes:          nodes,
		ch:             ch,
		overReadCH:     overReadCH,
		voteResults:    make(map[string]*VoteResult),
		voterRewards:   make(map[string]*VoterReward),
		coinBaseReward: make(map[string]uint64),
		period:         period,
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
		coinbaseTx, err := tx.GetCoinbaseTx(1200 * v.period)
		if err != nil {
			return err
		}
		for _, output := range coinbaseTx.Outputs {
			voteOutput, ok := output.TypedOutput.(*types.IntraChainOutput)
			if !ok {
				return errors.New("Output type error")
			}
			address := common.GetAddressFromControlProgram(voteOutput.ControlProgram)
			for _, node := range v.nodes {
				if address == node.MiningAddress {
					v.coinBaseReward[node.XPub] = voteOutput.Amount
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
			bigVoteNum = bigBlockNum.Mul(bigBlockNum, bigVoteNum)

			if value, ok := v.voteResults[voteInfo.XPub]; ok {
				value.Votes[voteInfo.Address] = bigVoteNum.Add(bigVoteNum, value.Votes[voteInfo.Address])
			} else {
				voteResult := &VoteResult{
					Votes:     make(map[string]*big.Int),
					VoteTotal: big.NewInt(0),
				}

				voteResult.Votes[voteInfo.Address] = bigVoteNum
				v.voteResults[voteInfo.XPub] = voteResult
			}

			v.voteResults[voteInfo.XPub].VoteTotal = bigVoteNum.Add(bigVoteNum, v.voteResults[voteInfo.XPub].VoteTotal)
		case <-v.overReadCH:
			break out
		}
	}
}

func (v *Vote) countReward() {
	for xpub, votes := range v.voteResults {
		coinbaseReward, ok := v.coinBaseReward[xpub]
		if !ok {
			log.Errorf("%s doesn't have a coinbase reward \n", xpub)
			continue
		}
		for address, vote := range votes.Votes {
			if value, ok := v.voterRewards[xpub]; ok {
				coinBaseReward := big.NewInt(0).SetUint64(coinbaseReward)
				value.rewards[address] = vote.Mul(vote, coinBaseReward).Div(vote, votes.VoteTotal)
			} else {
				reward := &VoterReward{
					rewards: make(map[string]*big.Int),
				}
				coinBaseReward := big.NewInt(0).SetUint64(coinbaseReward)
				reward.rewards[address] = vote.Mul(vote, coinBaseReward).Div(vote, votes.VoteTotal)
				v.voterRewards[xpub] = reward
			}
		}

	}
}

func (v *Vote) sendRewardTransaction() error {
	for _, node := range v.nodes {
		coinbaseReward, ok := v.coinBaseReward[node.XPub]
		if !ok {
			log.Errorf("%s doesn't have a coinbase reward \n", node.XPub)
			continue
		}

		if voterRewards, ok := v.voterRewards[node.XPub]; ok {
			txID, err := v.sendReward(coinbaseReward, node, voterRewards)
			if err != nil {
				return err
			}
			log.Info("tx_id: ", txID)
		}
	}

	return nil
}

func (v *Vote) sendReward(coinbaseReward uint64, node config.VoteRewardConfig, voterReward *VoterReward) (string, error) {
	var outputAction string

	inputAction := fmt.Sprintf(inputActionFmt, coinbaseReward, node.AccountID)

	index := 0
	for address, amount := range voterReward.rewards {
		outputAction += fmt.Sprintf(outputActionFmt, amount.Uint64(), address)
		index++
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
