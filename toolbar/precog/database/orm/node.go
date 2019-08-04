package orm

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vapor/toolbar/precog/common"
)

type Node struct {
	Alias           string
	PublicKey       string
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
	var lantency int64
	var activeMinutes uint64
	switch n.Status {
	case common.NodeHealthyStatus, common.NodeCongestedStatus, common.NodeOrphanStatus:
		lantency = n.LantencyMS.Int64
		activeMinutes = uint64(time.Since(n.ActiveBeginTime).Minutes())
	}

	return json.Marshal(&struct {
		Alias         string `json:"alias"`
		PublicKey     string `json:"public_key"`
		Host          string `json:"host"`
		Port          uint16 `json:"port"`
		BestHeight    uint64 `json:"best_height"`
		LantencyMS    int64  `json:"lantency_ms,omitempty"`
		ActiveMinutes uint64 `json:"active_minutes,omitempty"`
		Status        string `json:"status"`
	}{
		Alias:         n.Alias,
		PublicKey:     n.PublicKey,
		Host:          n.Host,
		Port:          n.Port,
		BestHeight:    n.BestHeight,
		LantencyMS:    lantency,
		ActiveMinutes: activeMinutes,
		Status:        status,
	})
}
