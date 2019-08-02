package orm

import (
	"time"
)

type Node struct {
	Alias      string       `json:"alias"`
	HostPort   string       `json:"host_port"`
	PubKey     chainkd.XPub `json:"pubkey"`
	BestHeight uint64       `json:"best_height"`
	LantencyMS uint64       `json:"lantency_ms"` // TODO:
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
