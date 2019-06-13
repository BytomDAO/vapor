package bc

import "io"

func (CancelVote) typ() string { return "cancelvote1" }
func (s *CancelVote) writeForHash(w io.Writer) {
	mustWriteForHash(w, s.SpentOutputId)
}

// SetDestination will link the spend to the output
func (s *CancelVote) SetDestination(id *Hash, val *AssetAmount, pos uint64) {
	s.WitnessDestination = &ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
	}
}

// NewCancelVote creates a new cancelVote.
func NewCancelVote(spentOutputID *Hash, ordinal uint64, vote []byte) *CancelVote {
	return &CancelVote{
		SpentOutputId: spentOutputID,
		Ordinal:       ordinal,
		Vote:          vote,
	}
}
