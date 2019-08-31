package orm

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vapor/toolbar/precog/common"
)

type Node struct {
	ID                       uint16 `gorm:"primary_key"`
	Alias                    string
	Xpub                     string
	PublicKey                string
	IP                       string
	Port                     uint16
	BestHeight               uint64
	AvgLantencyMS            sql.NullInt64
	LatestDailyUptimeMinutes uint64
	Status                   uint8
	CreatedAt                time.Time `json:"alias"`
	UpdatedAt                time.Time `json:"alias"`
}

func (n *Node) MarshalJSON() ([]byte, error) {
	status, ok := common.StatusLookupTable[n.Status]
	if !ok {
		return nil, errors.New("fail to look up status")
	}

	avgLantencyMS := uint64(0)
	if n.AvgLantencyMS.Valid {
		avgLantencyMS = uint64(n.AvgLantencyMS.Int64)
	}

	return json.Marshal(&struct {
		Alias                    string    `json:"alias"`
		PublicKey                string    `json:"publickey"`
		Address                  string    `json:"address"`
		BestHeight               uint64    `json:"best_height"`
		AvgLantencyMS            uint64    `json:"avg_lantency_ms"`
		LatestDailyUptimeMinutes uint64    `json:"latest_daily_uptime_minutes"`
		Status                   string    `json:"status"`
		UpdatedAt                time.Time `json:"updated_at"`
	}{
		Alias:                    n.Alias,
		PublicKey:                n.PublicKey,
		Address:                  fmt.Sprintf("%s:%d", n.IP, n.Port),
		BestHeight:               n.BestHeight,
		AvgLantencyMS:            avgLantencyMS,
		LatestDailyUptimeMinutes: n.LatestDailyUptimeMinutes,
		Status:    status,
		UpdatedAt: time.Now(),
	})
}
