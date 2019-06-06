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
	// db   *gorm.DB
	// node *service.Node
	blockKeeper         *blockKeeper
	unconfirmedTxKeeper *unconfirmedTxKeeper
}

func NewDbKeeper(db *gorm.DB, chainCfg *config.Chain) *DbKeeper {
	blockKeeper := &blockKeeper{
		db:   db,
		node: service.NewNode(chainCfg.Upstream.RPC),
	}

	unconfirmedTxKeeper := &unconfirmedTxKeeper{
		db:          db,
		processTxCh: make(chan *service.WSResponse, maxRawTxSize),
	}

	return &DbKeeper{blockKeeper, unconfirmedTxKeeper}
}

func (d *DbKeeper) Run() {
	go d.blockKeeper.Run()
	go d.unconfirmedTxKeeper.Run()
}
