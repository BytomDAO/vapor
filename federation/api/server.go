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
	v1.POST("/federation/list-transactions", handlerMiddleware(server.ListCrosschainTxs))

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

		c.Set(serverLabel, s)
		c.Next()
	}
}

// TODO:
func handlerMiddleware(handleFunc interface{}) func(*gin.Context) {
	// if err := common.ValidateFuncType(handleFunc); err != nil {
	// 	panic(err)
	// }

	return func(context *gin.Context) {
		handleRequest(context, handleFunc)
	}
}

// TODO: maybe move around
type handlerFun interface{}

// handleRequest get a handler function to process the request by request url
func handleRequest(context *gin.Context, fun handlerFun) {
	// args, err := buildHandleFuncArgs(fun, context)
	// if err != nil {
	// 	RespondErrorResp(context, err)
	// 	return
	// }

	// result := callHandleFunc(fun, args...)
	// if err := result[len(result)-1]; err != nil {
	// 	RespondErrorResp(context, err.(error))
	// 	return
	// }

	// if exist := processPaginationIfPresent(fun, args, result, context); exist {
	// 	return
	// }

	// if len(result) == 1 {
	// 	RespondSuccessResp(context, nil)
	// 	return
	// }

	// RespondSuccessResp(context, result[0])
}
