package orm

import (
	"database/sql"

	"time"
)

// TODO: json
type Node struct {
	Alias      string        `json:"alias"`
	PubKey     chainkd.XPub  `json:"pubkey"`
	HostPort   string        `json:"host_port"`
	BestHeight uint64        `json:"best_height"`
	LantencyMS sql.NullInt64 `json:"lantency_ms"`
	Status     uint8         `json:"status"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
