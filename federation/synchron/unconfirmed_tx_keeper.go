package synchron

import (
	// "time"

	// "github.com/bytom/errors"
	// "github.com/bytom/protocol/bc"
	// "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/federation/config"
	// "github.com/blockcenter/database"
	// "github.com/blockcenter/database/orm"
	"github.com/vapor/federation/service"
)

const maxRawTxSize = 1 << 16

type unconfirmedTxKeeper struct {
	cfg         *config.Chain
	db          *gorm.DB
	processTxCh chan *service.WSResponse
	// coinName    string
}

func (u *unconfirmedTxKeeper) Run() {
	ws := service.NewWSClient(u.cfg.Upstream.WebSocket, u.processTxCh)
	if err := ws.Connect(); err != nil {
		log.WithField("err", err).Fatal("websocket dail fail")
	}

	defer ws.Close()
	if err := ws.Subscribe(service.TopicNotifyNewTransactions); err != nil {
		log.WithField("err", err).Fatal("subscribe new transaction fail")
	}

	// u.receiveTransactions()
}
