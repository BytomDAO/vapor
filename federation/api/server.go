package api

import (
	"fmt"

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

	// r.Use(server.Middleware())
	// r.HEAD("/api/v1", handlerMiddleware(server.Head))
	// r.GET("/api/check-update", handlerMiddleware(server.CheckUpdate))

	// v1 := r.Group("/api/v1/:coin_name")

	// v1.POST("/merchant/build-payment", handlerMiddleware(server.BuildPayment))
	// v1.POST("/merchant/build-transaction", handlerMiddleware(server.BuildTransaction))
	// v1.POST("/merchant/submit-payment", handlerMiddleware(server.SubmitPayment))
	// v1.POST("/merchant/get-transaction", handlerMiddleware(server.GetTransaction))
	// v1.POST("/merchant/list-transactions", handlerMiddleware(server.ListTxs))
	// v1.POST("/merchant/create-txproposal", handlerMiddleware(server.CreateTxProposal))
	// v1.POST("/merchant/list-txproposals", handlerMiddleware(server.ListTxProposals))
	// v1.POST("/merchant/sign-txproposal", handlerMiddleware(server.SignTxProposal))
	// v1.POST("/merchant/submit-txproposal", handlerMiddleware(server.SubmitTxProposal))

	// v1.POST("/account/list-guids", handlerMiddleware(server.ListGuids))
	// v1.POST("/account/create", handlerMiddleware(server.CreateWallet))
	// v1.POST("/account/create-multisig", handlerMiddleware(server.CreateMultiSignWallet))
	// v1.POST("/account/join-multisig", handlerMiddleware(server.JoinMultiSignWallet))
	// v1.POST("/account/list-wallets", handlerMiddleware(server.ListWallets))
	// v1.POST("/account/new-address", handlerMiddleware(server.NewAddress))
	// v1.POST("/account/list-addresses", handlerMiddleware(server.ListAddress))
	// v1.POST("/account/restore", handlerMiddleware(server.RestoreWallet))

	// v1.GET("/q/chain-status", handlerMiddleware(server.ChainStatus))
	// v1.GET("/q/asset", handlerMiddleware(server.QueryAsset))
	// v1.POST("/q/list-utxos", handlerMiddleware(server.ListUtxos))

	// v1.GET("/q/slides", handlerMiddleware(server.GetSlides))
	// v1.GET("/q/apps", handlerMiddleware(server.GetApps))
	// v1.GET("/q/announces", handlerMiddleware(server.GetAnnounces))
	// v1.GET("/q/announce", handlerMiddleware(server.GetAnnounceDetail))
	// v1.GET("/q/explore", handlerMiddleware(server.GetExplorePage))

	server.engine = r
}

func (s *Server) Run() {
	s.engine.Run(fmt.Sprintf(":%d", s.cfg.API.ListeningPort))
}
