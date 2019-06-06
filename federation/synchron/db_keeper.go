package synchron

import (
	// "time"

	// "github.com/bytom/errors"
	// "github.com/bytom/protocol/bc"
	// "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"
	// log "github.com/sirupsen/logrus"

	"github.com/vapor/federation/config"
	// "github.com/blockcenter/database"
	// "github.com/blockcenter/database/orm"
	"github.com/vapor/federation/service"
)

type DbKeeper struct {
	blockKeeper *blockKeeper
}

func NewDbKeeper(db *gorm.DB, chainCfg *config.Chain) *DbKeeper {
	blockKeeper := &blockKeeper{
		cfg:  chainCfg,
		db:   db,
		node: service.NewNode(chainCfg.Upstream.RPC),
	}

	return &DbKeeper{blockKeeper}
}

func (d *DbKeeper) Run() {
	go d.blockKeeper.Run()
}
