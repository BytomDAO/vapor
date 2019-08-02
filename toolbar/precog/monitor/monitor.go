package monitor

import (
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/toolbar/precog/config"
)

type monitor struct {
	cfg *config.Config
	db  *gorm.DB
}

func NewMonitor(cfg *config.Config, db *gorm.DB) *monitor {
	return &monitor{
		cfg: cfg,
		db:  db,
	}
}

func (s *monitor) Run() {
	if err := s.updateNodesHostPort(); err != nil {
		// TODO: redirect output to logfile
		log.Fatal(err)
	}

	// for ticker, dail nodes
}

func (s *monitor) updateNodesHostPort() error {
	return nil
}
