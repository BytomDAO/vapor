package synchron

import (
	"github.com/jinzhu/gorm"

	"github.com/vapor/federation/config"
	"github.com/vapor/federation/service"
)

type mainchainKeeper struct {
	cfg       *config.Chain
	db        *gorm.DB
	node      *service.Node
	chainName string
}

func NewMainchainKeeper(db *gorm.DB, chainCfg *config.Chain) *mainchainKeeper {
	return &mainchainKeeper{
		cfg:       chainCfg,
		db:        db,
		node:      service.NewNode(chainCfg.Upstream),
		chainName: chainCfg.Name,
	}
}

func (m *mainchainKeeper) Run() {}
