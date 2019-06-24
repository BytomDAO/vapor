package api

import (
	"github.com/jinzhu/gorm"

	"github.com/vapor/federation/config"
)

type Server struct {
	cfg *config.Config
	db  *gorm.DB
}

func NewServer(db *gorm.DB, cfg *config.Config) *Server {
	server := &Server{
		cfg: cfg,
		db:  db,
	}
	// setupRouter(server)
	return server
}

func (s *Server) Run() {

}
