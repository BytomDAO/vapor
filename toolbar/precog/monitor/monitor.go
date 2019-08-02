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

func (m *monitor) Run() {
	if err := m.updateBootstrapNodes(); err != nil {
		log.Fatal(err)
	}

	ticker := time.NewTicker(checkFreqSeconds * time.Second)
	for ; true; <-ticker.C {
		// TODO: lock?
		m.monitorRountine()
	}
}

func (m *monitor) updateBootstrapNodes() error {
	var existedNodes, newNodes []config.Node

	// TODO: use affected comlumns?
	for _, node := range m.cfg.Nodes {
		if true {
			existedNodes = append(existedNodes, node)
		} else {
			newNodes = append(newNodes, node)
		}
	}

	return nil
}

func (m *monitor) monitorRountine() error {
	// dail
	// get blockhash
	// update
	return nil
}
