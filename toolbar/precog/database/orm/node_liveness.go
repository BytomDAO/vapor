package orm

import (
	"database/sql"
	"time"
)

type NodeLiveness struct {
	ID            uint64 `gorm:"primary_key"`
	NodeID        uint16
	PingTimes     uint64
	PongTimes     uint64
	AvgLantencyMS sql.NullInt64
	BestHeight    uint64
	// Status        uint8
	CreatedAt time.Time
	UpdatedAt time.Time

	Node *Node `gorm:"foreignkey:NodeID"`
}
