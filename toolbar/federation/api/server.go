package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"github.com/bytom/vapor/toolbar/federation/config"
	serverCommon "github.com/bytom/vapor/toolbar/server"
)

type Server struct {
	cfg    *config.Config
	db     *gorm.DB
	engine *gin.Engine
}

func NewServer(db *gorm.DB, cfg *config.Config) *Server {
	server := &Server{
		cfg: cfg,
		db:  db,
	}
	if cfg.API.IsReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}
	server.setupRouter()
	return server
}

func (s *Server) setupRouter() {
	r := gin.Default()
	r.Use(serverCommon.Middleware(s))

	v1 := r.Group("/api/v1")
	v1.POST("/federation/list-crosschain-txs", serverCommon.HandlerMiddleware(s.ListCrosschainTxs))
	v1.GET("/federation/list-chains", serverCommon.HandlerMiddleware(s.ListChains))

	s.engine = r
}

func (s *Server) Run() {
	s.engine.Run(":9886")
}
