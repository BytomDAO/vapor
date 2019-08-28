package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"github.com/vapor/toolbar/federation/config"
	"github.com/vapor/toolbar/server"
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
	r.Use(server.Middleware(s))

	v1 := r.Group("/api/v1")
	v1.POST("/federation/list-crosschain-txs", server.HandlerMiddleware(s.ListCrosschainTxs))
	v1.GET("/federation/list-chains", server.HandlerMiddleware(s.ListChains))

	s.engine = r
}

func (s *Server) Run() {
	s.engine.Run(":9886")
}
