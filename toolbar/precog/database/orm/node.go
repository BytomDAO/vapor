package orm

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Node struct {
	Alias           string        `json:"alias"`
	PubKey          chainkd.XPub  `json:"pubkey"`
	Host            string        `json:"host"`
	Port            uint16        `json:"port"`
	BestHeight      uint64        `json:"best_height"`
	LantencyMS      sql.NullInt64 `json:"lantency_ms"`
	ActiveBeginTime time.Time     `json:"active_begin_time"`
	Status          uint8         `json:"status"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TODO:
func (n *Node) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Alias      string        `json:"alias"`
		PubKey     chainkd.XPub  `json:"pubkey"`
		Host       string        `json:"host"`
		Port       uint16        `json:"port"`
		BestHeight uint64        `json:"best_height"`
		LantencyMS sql.NullInt64 `json:"lantency_ms,omitempty"`
		ActiveTime time.Duration `json:"active_time,omitempty"`
		Status     string        `json:"status"`
	}{
		Alias:      n.Alias,
		PubKey:     n.PubKey,
		Host:       n.Host,
		Port:       n.Port,
		BestHeight: n.BestHeight,
	})
}
