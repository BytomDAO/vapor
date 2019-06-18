package orm

import (
	"github.com/vapor/federation/types"
)

type Warder struct {
	// WarderID has to be the same as its position
	ID        uint64
	Pubkey    string
	CreatedAt types.Timestamp
	UpdatedAt types.Timestamp
}
