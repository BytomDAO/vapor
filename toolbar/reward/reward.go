package reward

import (
	"github.com/jinzhu/gorm"

	"github.com/vapor/toolbar/reward/config"
	"github.com/vapor/toolbar/reward/database/orm"
	instance "github.com/vapor/toolbar/reward/reward"
)

type CountReward interface {
	Start()
}

type Reward struct {
	cfg         *config.Config
	db          *gorm.DB
	countReward CountReward
	VoteInfoCh  chan instance.VoteInfo
	OverReadCh  chan struct{}
	period      uint64
}

func NewReward(db *gorm.DB, cfg *config.Config, period uint64) *Reward {
	voteInfoCh := make(chan instance.VoteInfo)
	overReadCh := make(chan struct{})
	var countReward CountReward
	if len(cfg.VoteConf) != 0 {
		countReward = instance.NewVote(cfg.VoteConf, voteInfoCh, overReadCh, period)
	} else if cfg.OptionalNodeConf != nil {
		// OptionalNode reward instance
	}

	if countReward == nil {
		panic("There are no instances of rewards being handed out, please check the configuration")
	}

	reward := &Reward{
		cfg:         cfg,
		db:          db,
		countReward: countReward,
		VoteInfoCh:  voteInfoCh,
		OverReadCh:  overReadCh,
		period:      period,
	}

	return reward
}

func (r *Reward) readVoteInfo() error {
	xpubs := []string{}
	for _, node := range r.cfg.VoteConf {
		xpubs = append(xpubs, node.XPub)
	}

	minHeight := (1 + 1200*(r.period-1))
	maxHeight := 1200 * r.period
	rows, err := r.db.Model(&orm.Utxo{}).Select("xpub, voter_address, vote_num, vote_height, veto_height").Where("vote_height BETWEEN ? and ? and xpub in (?)", minHeight, maxHeight, xpubs).Rows()
	if err != nil {
		return err
	}

	for rows.Next() {
		var (
			xpub       string
			address    string
			voteNum    uint64
			voteHeight uint64
			vetoHeight uint64

			voteBlockNum uint64
		)
		if err := rows.Scan(&xpub, &address, &voteNum, &voteHeight, &vetoHeight); err != nil {
			return err
		}

		if vetoHeight > 0 && vetoHeight < 1200*r.period {
			voteBlockNum = vetoHeight - voteHeight
		} else {
			voteBlockNum = 1200*r.period - voteHeight
		}
		r.VoteInfoCh <- instance.VoteInfo{
			XPub:         xpub,
			Address:      address,
			VoteNum:      voteNum,
			VoteHeight:   voteHeight,
			VetoHeight:   vetoHeight,
			VoteBlockNum: voteBlockNum,
		}
	}

	close(r.OverReadCh)
	return nil
}

func (r *Reward) Start() {
	go r.readVoteInfo()
	go r.countReward.Start()
}
