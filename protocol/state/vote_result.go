package state

import (
	"github.com/vapor/protocol/bc"
)

// VoteResult represents a snapshot of each round of DPOS voting
// Seq indicates the sequence of current votes, which start from zero
// NumOfVote indicates the number of votes each consensus node receives, the key of map represent public key
// Finalized indicates whether this vote is finalized
type VoteResult struct {
	Seq             uint64
	NumOfVote       map[string]uint64
	LastBlockHash   bc.Hash
	Finalized       bool
}
