package api

import (
	"github.com/jinzhu/gorm"

	"github.com/vapor/toolbar/precog/config"
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
	// setup gin

	// disable log

	// bind handle
}
