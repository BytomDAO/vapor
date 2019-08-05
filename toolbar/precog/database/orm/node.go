package orm

import (
	"time"
)

type Node struct {
	Alias      string
	PublicKey  string
	Host       string
	Port       uint16
	BestHeight uint64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
