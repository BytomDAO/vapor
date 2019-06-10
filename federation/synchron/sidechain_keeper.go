package synchron

import (
	"github.com/jinzhu/gorm"

	"github.com/vapor/federation/config"
	"github.com/vapor/federation/service"
)

type sidechainKeeper struct {
	cfg       *config.Chain
	db        *gorm.DB
	node      *service.Node
	chainName string
}

func NewSidechainKeeper(db *gorm.DB, chainCfg *config.Chain) *sidechainKeeper {
	return &sidechainKeeper{
		cfg:       chainCfg,
		db:        db,
		node:      service.NewNode(chainCfg.Upstream),
		chainName: chainCfg.Name,
	}
}

func (s *sidechainKeeper) Run() {}
