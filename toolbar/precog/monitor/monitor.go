package monitor

import (
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/toolbar/precog/config"
)

// TODO: put in cfg?
const checkFreqSeconds = 60

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
		log.Fatal(err)
	}

	ticker := time.NewTicker(checkFreqSeconds * time.Second)
	for ; true; <-ticker.C {
	}
}

func (s *monitor) updateNodesHostPort() error {
	return nil
}
