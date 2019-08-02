package monitor

import (
	"github.com/jinzhu/gorm"

	log "github.com/sirupsen/logrus"
)

type monitor struct {
	db *gorm.DB
}

func NewMonitor(db *gorm.DB) *monitor {
	return &monitor{db: db}
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
