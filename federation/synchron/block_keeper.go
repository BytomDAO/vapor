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

type blockKeeper struct {
	cfg  *config.Chain
	db   *gorm.DB
	node *service.Node
	// cache    *database.RedisDB
	// coinName string
}

func NewBlockKeeper(db *gorm.DB, chainCfg *config.Chain) *blockKeeper {
	return &blockKeeper{
		cfg:  chainCfg,
		db:   db,
		node: service.NewNode(chainCfg.Upstream.RPC),
	}
}

func (b *blockKeeper) Run() {

}
