package state

import (
	// "github.com/vapor/consensus"
	// "github.com/vapor/database/storage"
	// "github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
)

// MainchainOutputView represents a view into the set of manchain outputs
type MainchainOutputViewpoint struct {
	Entries map[bc.Hash]MainchainOutputEntry
}

type MainchainOutputEntry struct {
	Claimed bool `json:"claimed,omitempty"`
}
