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
	node               *config.VoteRewardConfig
	db                 *gorm.DB
	reward             *voterReward
	coinBaseRewards    map[uint64]*coinBaseReward
	period             uint64
	roundVoteBlockNums uint64
	rewardStartHeight  uint64
	rewardEndHeight    uint64
}

func NewVote(db *gorm.DB, node *config.VoteRewardConfig, rewardStartHeight, rewardEndHeight uint64) *Vote {
	return &Vote{
		db:                 db,
		node:               node,
		reward:             &voterReward{rewards: make(map[string]*big.Int)},
		coinBaseRewards:    make(map[uint64]*coinBaseReward),
		roundVoteBlockNums: consensus.ActiveNetParams.DPOSConfig.RoundVoteBlockNums,
		rewardStartHeight:  rewardStartHeight,
		rewardEndHeight:    rewardEndHeight,
	}
}

func (v *Vote) getVoteByXpub(height uint64) ([]*orm.Utxo, error) {
	utxos := []*orm.Utxo{}
	if err := v.db.Where("(veto_height >= ? or veto_height = 0) and vote_height <= ? and xpub = ?", height-v.roundVoteBlockNums+1, height-v.roundVoteBlockNums, v.node.XPub).Find(&utxos).Error; err != nil {
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
	for height := v.rewardStartHeight + v.roundVoteBlockNums; height <= v.rewardEndHeight; height += v.roundVoteBlockNums {
		voteInfo, err := v.getVoteByXpub(height)
		if err != nil {
			log.WithFields(log.Fields{"error": err, "coinbase_height": height}).Error("get vote from db")
			continue
		}

		voteResults := v.countVotes(voteInfo, height)
		v.countReward(voteResults, height)
	}
	// send transactions
	v.sendRewardTransaction()

}

func (v *Vote) getCoinbaseReward() error {

	tx := Transaction{
		ip: fmt.Sprintf("http://%s:%d", v.node.Host, v.node.Port),
	}
	for height := v.rewardStartHeight + v.roundVoteBlockNums; height <= v.rewardEndHeight; height += v.roundVoteBlockNums {
		coinbaseTx, err := tx.GetCoinbaseTx(height)
		if err != nil {
			log.WithFields(log.Fields{"error": err, "coinbase_height": height}).Error("get coinbase reward")
			return err
		}
	out:
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

			if address == v.node.MiningAddress {
				reward := &coinBaseReward{
					totalReward: output.Amount,
				}
				ratioNumerator := big.NewInt(int64(v.node.RewardRatio))
				ratioDenominator := big.NewInt(100)
				coinBaseReward := big.NewInt(0).SetUint64(output.Amount)
				reward.voteTotalReward = coinBaseReward.Mul(coinBaseReward, ratioNumerator).Div(coinBaseReward, ratioDenominator)
				v.coinBaseRewards[height] = reward
				break out
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

func (v *Vote) countReward(votes *voteResult, height uint64) {
	coinBaseReward, ok := v.coinBaseRewards[height]
	if !ok {
		log.Errorf("Doesn't have a coinbase reward")
		return
	}

	for address, vote := range votes.votes {
		if reward, ok := v.reward.rewards[address]; ok {
			mul := vote.Mul(vote, coinBaseReward.voteTotalReward)
			amount := big.NewInt(0)
			amount.Div(mul, votes.voteTotal)
			reward.Add(reward, amount)
		} else {
			mul := vote.Mul(vote, coinBaseReward.voteTotalReward)
			amount := big.NewInt(0)
			amount.Div(mul, votes.voteTotal)
			if amount.Uint64() > 0 {
				v.reward.rewards[address] = amount
				v.reward.totalCoinbaseReward = coinBaseReward.totalReward
				v.reward.height = height
			}
		}
	}
}

func (v *Vote) sendRewardTransaction() {
	txID, err := v.sendReward()
	if err != nil {
		log.WithFields(log.Fields{"error": err, "node": v.node}).Error("send reward transaction")
		return
	}
	log.Info("tx_id: ", txID)

}

func (v *Vote) sendReward() (string, error) {
	var outputAction string
	inputAction := fmt.Sprintf(inputActionFmt, v.reward.totalCoinbaseReward, v.node.AccountID)
	index := 0
	for address, amount := range v.reward.rewards {
		index++
		outputAction += fmt.Sprintf(outputActionFmt, amount.Uint64(), address)
		if index != len(v.reward.rewards) {
			outputAction += ","
		}
	}
	tx := Transaction{
		ip: fmt.Sprintf("http://%s:%d", v.node.Host, v.node.Port),
	}

	tmpl, err := tx.buildTx(inputAction, outputAction)
	if err != nil {
		return "", err
	}

	tmpl, err = tx.signTx(v.node.Passwd, *tmpl)
	if err != nil {
		return "", err
	}

	return tx.SubmitTx(tmpl.Transaction)
}
