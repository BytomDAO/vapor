package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"github.com/vapor/toolbar/precog/config"
	"github.com/vapor/toolbar/precog/database/orm"
)

type server struct {
	cfg *config.Config
	db  *gorm.DB
}

func NewApiServer(cfg *config.Config, db *gorm.DB) *server {
	return &server{
		cfg: cfg,
		db:  db,
	}
}

func (s *server) Run() {
	router := gin.Default()

	// router.POST("/list-nodes", listNodes)

	router.Run(fmt.Sprintf(":%d", s.cfg.API.ListeningPort))
}

// TODO: cannot use listNodes (type func(*gin.Context) ([]*orm.Node, error)) as type gin.HandlerFunc in argument to router.RouterGroup.POST
func listNodes(_ *gin.Context) ([]*orm.Node, error) {
	return nil, nil
}
