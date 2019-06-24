package api

import (
	"github.com/vapor/federation/config"
)

type Server struct {
	// db         *database.DB
	// cache      database.Cache
	// node       Node
	// cfgManager *service.DBConfigManager
	// cfg        *config.Config
	// engine     *gin.Engine
}

func NewServer(cfg *config.Config) *Server {
	//     db, err := database.NewMySQLDB(cfg.MySQL, cfg.API.MySQLConnCfg)
	//     if err != nil {
	//         log.WithField("err", err).Panic("initialize mysql db error")
	//     }

	//     cache, err := database.NewRedisDB(cfg.Redis)
	//     if err != nil {
	//         log.WithField("err", err).Panic("initialize redis error")
	//     }

	//     node := service.NewBytomNode(cfg.Coin.Btm.Upstream.URL)
	//     return NewServerWithPersistenceAndNode(cfg, db, cache, node)
	return nil
}

func (s *Server) Run() {

}
