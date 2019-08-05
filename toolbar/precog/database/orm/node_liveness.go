package orm

import (
	"database/sql"
	"time"
)

type NodeLiveness struct {
	NodeID        uint16
	ProbeTimes    uint64
	ResponseTimes uint64
	AvgLantencyMS sql.NullInt64
	Status        uint8
	CreatedAt     time.Time
	UpdatedAt     time.Time

	Node *Node `gorm:"foreignkey:NodeID"`
}
