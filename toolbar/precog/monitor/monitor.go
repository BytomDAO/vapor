package monitor

import (
	"github.com/jinzhu/gorm"
)

type monitor struct {
	db *gorm.DB
}

func NewMonitor(db *gorm.DB) *monitor {
	return &monitor{db: db}
}

func (s *monitor) Run() {
}
