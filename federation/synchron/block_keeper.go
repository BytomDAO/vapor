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

type blockKeeper struct {
	// cfg      *config.Config
	db *gorm.DB
	// cache    *database.RedisDB
	node *service.Node
	// coinName string
}

func (b *blockKeeper) Run() {

}
