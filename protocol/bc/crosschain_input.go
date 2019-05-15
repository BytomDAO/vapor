package bc

import "io"

// crosschaininput is the result of a transfer of value. The value it contains
// comes from the main chain. It satisfies the Entry interface.

func (CrossChainInput) typ() string { return "crosschaininput1" }

func (s *CrossChainInput) writeForHash(w io.Writer) {
	mustWriteForHash(w, s.MainChainOutputId)
}

// SetDestination will link the CrossChainInput to the output
func (s *CrossChainInput) SetDestination(id *Hash, val *AssetAmount, pos uint64) {
	s.WitnessDestination = &ValueDestination{
		Ref:      id,
		Value:    val, //???
		Position: pos,
	}
}

// NewCrossChainInput creates a new CrossChainInput.
func NewCrossChainInput(mainChainOutputID *Hash, ordinal uint64) *CrossChainInput {
	return &CrossChainInput{
		MainChainOutputId: mainChainOutputID,
		Ordinal:           ordinal,
	}
}
