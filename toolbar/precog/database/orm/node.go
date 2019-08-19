package orm

import (
	"time"
)

type Node struct {
	ID                       uint16 `gorm:"primary_key"`
	Alias                    string
	Xpub                     string
	PublicKey                string
	Host                     string
	Port                     uint16
	BestHeight               uint64
	LatestDailyUptimeMinutes uint64
	CreatedAt                time.Time
	UpdatedAt                time.Time
}
