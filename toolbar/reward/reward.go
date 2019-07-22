package reward

import (
	"github.com/jinzhu/gorm"

	"github.com/vapor/toolbar/reward/config"
	instance "github.com/vapor/toolbar/reward/rewardinstance"
)

type CountReward interface {
	Start() error
}

type Reward struct {
	countReward CountReward
}

func NewReward(db *gorm.DB, cfg *config.Config, rewardStartHeight, rewardEndHeight uint64) *Reward {
	var countReward CountReward
	if cfg.VoteConf != nil {
		countReward = instance.NewVote(db, cfg.VoteConf, rewardStartHeight, rewardEndHeight)
	}

	if countReward == nil {
		panic("There are no instances of rewards being handed out, please check the configuration")
	}

	return &Reward{
		countReward: countReward,
	}

}

func (r *Reward) Start() error {
	return r.countReward.Start()
}
