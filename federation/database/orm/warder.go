package orm

import (
	"github.com/vapor/federation/types"
)

type Warder struct {
	ID        uint64
	Pubkey    string
	CreatedAt types.Timestamp
	UpdatedAt types.Timestamp
}
