package synchron

import (
	// "time"

	// "github.com/bytom/errors"
	// "github.com/bytom/protocol/bc"
	// "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"
	// log "github.com/sirupsen/logrus"

	// "github.com/vapor/federation/config"
	// "github.com/blockcenter/database"
	// "github.com/blockcenter/database/orm"
	"github.com/vapor/federation/service"
)

const maxRawTxSize = 1 << 16

type unconfirmedTxKeeper struct {
	// cfg         *config.Config
	db *gorm.DB
	// coinName    string
	processTxCh chan *service.WSResponse
}

func (u *unconfirmedTxKeeper) Run() {

}
