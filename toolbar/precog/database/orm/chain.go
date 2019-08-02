package orm

import (
	"time"
)

type Chain struct {
	BestHeight uint64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
