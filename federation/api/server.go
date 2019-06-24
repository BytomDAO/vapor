package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"github.com/vapor/federation/config"
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
	setupRouter(server)
	return server
}

func setupRouter(server *Server) {
	r := gin.Default()

	r.Use(server.Middleware())
	v1 := r.Group("/api/v1")

	v1.POST("/federation/list-transactions", handlerMiddleware(server.ListTxs))

	server.engine = r
}

func (s *Server) Run() {
	s.engine.Run(fmt.Sprintf(":%d", s.cfg.API.ListeningPort))
}

func (s *Server) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// add Access-Control-Allow-Origin
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

func handlerMiddleware(handleFunc interface{}) func(*gin.Context) {
	return nil
	// if err := common.ValidateFuncType(handleFunc); err != nil {
	// 	panic(err)
	// }

	// return func(context *gin.Context) {
	// 	server := context.MustGet(common.ServerLabel).(*Server)
	// 	banned, err := server.isBannedIP(context.Request.RemoteAddr)
	// 	if err != nil {
	// 		common.RespondErrorResp(context, err)
	// 		return
	// 	}

	// 	if banned {
	// 		common.RespondErrorResp(context, types.ErrBannedIPOrWallet)
	// 		return
	// 	}

	// 	coin, err := server.QueryCoinByName(context.Param("coin_name"))
	// 	if err != nil {
	// 		common.RespondErrorResp(context, err)
	// 		return
	// 	}

	// 	context.Set(common.CoinLabel, coin)
	// 	common.HandleRequest(context, handleFunc)
	// }
}
