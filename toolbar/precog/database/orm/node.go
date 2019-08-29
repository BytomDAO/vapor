package orm

import (
	"encoding/json"
	"fmt"
	"time"
)

type Node struct {
	ID                       uint16 `gorm:"primary_key"`
	Alias                    string
	Xpub                     string
	PublicKey                string
	IP                       string
	Port                     uint16
	BestHeight               uint64
	LatestDailyUptimeMinutes uint64
	Status                   uint8
	CreatedAt                time.Time `json:"alias"`
	UpdatedAt                time.Time `json:"alias"`
}

func (n *Node) MarshalJSON() ([]byte, error) {
	var status string
	// TODO:
	// switch c.Status {
	// case common.CrossTxPendingStatus:
	// 	status = common.CrossTxPendingStatusLabel
	// case common.CrossTxCompletedStatus:
	// 	status = common.CrossTxCompletedStatusLabel
	// default:
	// 	return nil, errors.New("unknown cross-chain tx status")
	// }

	return json.Marshal(&struct {
		Alias                    string    `json:"alias"`
		PublicKey                string    `json:"publickey"`
		Address                  string    `json:"address"`
		BestHeight               uint64    `json:"best_height"`
		LatestDailyUptimeMinutes uint64    `json:"latest_daily_uptime_minutes"`
		Status                   string    `json:"status"`
		UpdatedAt                time.Time `json:"updated_at"`
	}{
		Alias:                    n.Alias,
		PublicKey:                n.PublicKey,
		Address:                  fmt.Sprintf("%s:%d", n.IP, n.Port),
		BestHeight:               n.BestHeight,
		LatestDailyUptimeMinutes: n.LatestDailyUptimeMinutes,
		Status: status,
		// TODO:
		UpdatedAt: time.Now(),
	})
}
