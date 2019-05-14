package state

// VoteResult represents a snapshot of each round of DPOS voting
// Seq indicates the sequence of current votes, which start from zero
// NumOfVote indicates the number of votes each consensus node receives, the key of map represent public key
// LastBlockHeight indicates the last voted block height
// Finalized indicates whether this vote is finalized
type VoteResult struct {
	Seq             uint64
	NumOfVote       map[string]uint64
	LastBlockHeight uint64
	Finalized       bool
}
