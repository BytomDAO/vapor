package reward

import (
	"github.com/jinzhu/gorm"

	"github.com/vapor/consensus"
	"github.com/vapor/toolbar/reward/config"
	"github.com/vapor/toolbar/reward/database/orm"
	instance "github.com/vapor/toolbar/reward/rewardinstance"
)

type CountReward interface {
	Start([]*orm.Utxo)
}

type Reward struct {
	cfg                *config.Config
	db                 *gorm.DB
	countReward        CountReward
	rewardStartHeight  uint64
	rewardEndHeight    uint64
	roundVoteBlockNums uint64
}

func NewReward(db *gorm.DB, cfg *config.Config, rewardStartHeight, rewardEndHeight uint64) *Reward {
	var countReward CountReward
	if len(cfg.VoteConf) != 0 {
		countReward = instance.NewVote(cfg.VoteConf, rewardStartHeight, rewardEndHeight)
	} else if cfg.OptionalNodeConf != nil {
		// OptionalNode reward instance
	}

	if countReward == nil {
		panic("There are no instances of rewards being handed out, please check the configuration")
	}

	reward := &Reward{
		cfg:                cfg,
		db:                 db,
		countReward:        countReward,
		rewardStartHeight:  rewardStartHeight,
		rewardEndHeight:    rewardEndHeight,
		roundVoteBlockNums: consensus.ActiveNetParams.DPOSConfig.RoundVoteBlockNums,
	}

	return reward
}

func (r *Reward) sendReward() error {
	xpubs := []string{}
	for _, node := range r.cfg.VoteConf {
		xpubs = append(xpubs, node.XPub)
	}

	utxos := []*orm.Utxo{}
	if err := r.db.Where("(veto_height >= ? or veto_height = 0) and vote_height <= ? and xpub in (?)", r.rewardStartHeight+1, r.rewardEndHeight, xpubs).Find(&utxos).Error; err != nil {
		return err
	}

	r.countReward.Start(utxos)
	return nil
}

func (r *Reward) Start() {
	r.sendReward()
}
