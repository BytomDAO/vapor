package orm

import (
	"time"
)

type Node struct {
	ID                       uint16    `gorm:"primary_key"`
	Alias                    string    `json:"alias"`
	Xpub                     string    `json:"-"`
	PublicKey                string    `json:"publickey"`
	IP                       string    `json:"ip"`
	Port                     uint16    `json:"port"`
	BestHeight               uint64    `json:"alias"`
	LatestDailyUptimeMinutes uint64    `json:"alias"`
	CreatedAt                time.Time `json:"alias"`
	UpdatedAt                time.Time `json:"alias"`
}
