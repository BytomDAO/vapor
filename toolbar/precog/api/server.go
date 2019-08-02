package api

import (
	"github.com/jinzhu/gorm"
)

type server struct {
	db *gorm.DB
}

func NewApiServer(db *gorm.DB) *server {
	return &server{db: db}
}

func (s *server) Run() {
	// setup gin

	// disable log

	// bind handle
}
