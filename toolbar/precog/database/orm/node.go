package orm

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vapor/toolbar/precog/common"
)

type Node struct {
	Alias           string
	PubKey          chainkd.XPub
	Host            string
	Port            uint16
	BestHeight      uint64
	LantencyMS      sql.NullInt64
	ActiveBeginTime time.Time
	Status          uint8
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (n *Node) MarshalJSON() ([]byte, error) {
	status := common.StatusMap[n.Status]
	var lantency uint64
	var activeTime time.Duration
	if n.Status != common.NodeOfflineStatus {
		lantency = n.LantencyMS
		activeTime = time.Since(n.ActiveBeginTime)
	}

	return json.Marshal(&struct {
		Alias      string        `json:"alias"`
		PubKey     chainkd.XPub  `json:"pubkey"`
		Host       string        `json:"host"`
		Port       uint16        `json:"port"`
		BestHeight uint64        `json:"best_height"`
		LantencyMS uint64        `json:"lantency_ms,omitempty"`
		ActiveTime time.Duration `json:"active_time,omitempty"`
		Status     string        `json:"status"`
	}{
		Alias:      n.Alias,
		PubKey:     n.PubKey,
		Host:       n.Host,
		Port:       n.Port,
		BestHeight: n.BestHeight,
		LantencyMS: lantency,
		activeTime: activeTime,
		Status:     status,
	})
}
