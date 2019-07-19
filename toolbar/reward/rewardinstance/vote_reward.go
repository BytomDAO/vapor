package reward

import (
	"fmt"
	"math/big"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/consensus"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/toolbar/common"
	"github.com/vapor/toolbar/reward/config"
	"github.com/vapor/toolbar/reward/database/orm"
)

type voterReward struct {
	rewards             map[string]*big.Int
	totalCoinbaseReward uint64
	height              uint64
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
	db                 *gorm.DB
	voterRewards       map[string]*voterReward
	coinBaseRewards    map[string]*coinBaseReward
	period             uint64
	roundVoteBlockNums uint64
	rewardStartHeight  uint64
	rewardEndHeight    uint64
}

func NewVote(db *gorm.DB, nodes []config.VoteRewardConfig, rewardStartHeight, rewardEndHeight uint64) *Vote {
	return &Vote{
		db:                 db,
		nodes:              nodes,
		voterRewards:       make(map[string]*voterReward),
		coinBaseRewards:    make(map[string]*coinBaseReward),
		roundVoteBlockNums: consensus.ActiveNetParams.DPOSConfig.RoundVoteBlockNums,
		rewardStartHeight:  rewardStartHeight,
		rewardEndHeight:    rewardEndHeight,
	}
}

func (v *Vote) getVoteByXpub(xpub string, height uint64) ([]*orm.Utxo, error) {
	utxos := []*orm.Utxo{}
	if err := v.db.Where("(veto_height >= ? or veto_height = 0) and vote_height <= ? and xpub = ?", height-v.roundVoteBlockNums+1, height-v.roundVoteBlockNums, xpub).Find(&utxos).Error; err != nil {
		return nil, err
	}
	return utxos, nil
}

func (v *Vote) Start() {
	// get coinbase reward
	err := v.getCoinbaseReward()
	if err != nil {
		return
	}
	for _, node := range v.nodes {
		for height := v.rewardStartHeight + v.roundVoteBlockNums; height <= v.rewardEndHeight; height += v.roundVoteBlockNums {
			voteInfo, err := v.getVoteByXpub(node.XPub, height)
			if err != nil {
				log.WithFields(log.Fields{"error": err, "coinbase_height": height}).Error("get vote from db")
				continue
			}

			voteResults := v.countVotes(voteInfo, height)
			v.countReward(voteResults, node.XPub, height)
		}
	}
	// send transactions
	v.sendRewardTransaction()

}

func (v *Vote) getCoinbaseReward() error {
	if len(v.nodes) > 0 {
		tx := Transaction{
			ip: fmt.Sprintf("http://%s:%d", v.nodes[0].Host, v.nodes[0].Port),
		}
		for height := v.rewardStartHeight + v.roundVoteBlockNums; height <= v.rewardEndHeight; height += v.roundVoteBlockNums {
			coinbaseTx, err := tx.GetCoinbaseTx(height)
			if err != nil {
				log.WithFields(log.Fields{"error": err, "coinbase_height": height}).Error("get coinbase reward")
				return err
			}
			for _, output := range coinbaseTx.Outputs {
				output, ok := output.TypedOutput.(*types.IntraChainOutput)
				if !ok {
					log.WithFields(log.Fields{"error": err, "coinbase_height": height}).Error("Output type error")
					return errors.New("Output type error")
				}
				address := common.GetAddressFromControlProgram(output.ControlProgram)
				if output.Amount == 0 {
					continue
				}
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
						key := fmt.Sprintf("%s_%d", node.XPub, height)
						v.coinBaseRewards[key] = reward
						break out
					}
				}
			}
		}
	}
	return nil
}

func (v *Vote) countVotes(utxos []*orm.Utxo, coinBaseHeight uint64) (voteResults *voteResult) {
	voteResults = &voteResult{
		votes:     make(map[string]*big.Int),
		voteTotal: big.NewInt(0),
	}
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

		if vote, ok := voteResults.votes[utxo.VoterAddress]; ok {
			vote.Add(vote, bigVoteNum)
		} else {
			voteResults.votes[utxo.VoterAddress] = bigVoteNum
		}

		voteTotal := voteResults.voteTotal
		voteTotal.Add(voteTotal, bigVoteNum)
		voteResults.voteTotal = voteTotal
	}
	return
}

func (v *Vote) countReward(votes *voteResult, xpub string, height uint64) {
	key := fmt.Sprintf("%s_%d", xpub, height)
	coinBaseReward, ok := v.coinBaseRewards[key]
	if !ok {
		log.Errorf("%s doesn't have a coinbase reward \n", xpub)
		return
	}

	for address, vote := range votes.votes {
		if value, ok := v.voterRewards[xpub]; ok {
			mul := vote.Mul(vote, coinBaseReward.voteTotalReward)
			amount := big.NewInt(0)
			amount.Div(mul, votes.voteTotal)
			value.rewards[address] = amount
			if value.height < height {
				value.totalCoinbaseReward += coinBaseReward.totalReward
				value.height = height

			}
		} else {
			reward := &voterReward{
				rewards: make(map[string]*big.Int),
			}

			mul := vote.Mul(vote, coinBaseReward.voteTotalReward)
			amount := big.NewInt(0)
			amount.Div(mul, votes.voteTotal)
			if amount.Uint64() > 0 {
				reward.rewards[address] = amount
				reward.totalCoinbaseReward = coinBaseReward.totalReward
				reward.height = height
				v.voterRewards[xpub] = reward
			}
		}
	}
}

func (v *Vote) sendRewardTransaction() {
	for _, node := range v.nodes {
		if voterReward, ok := v.voterRewards[node.XPub]; ok {
			txID, err := v.sendReward(voterReward.totalCoinbaseReward, node, voterReward)
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
