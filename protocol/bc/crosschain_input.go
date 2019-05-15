package bc

import "io"

// crosschaininput is the result of a transfer of value. The value it contains
// comes from the main chain. It satisfies the Entry interface.

func (CrossChainInput) typ() string { return "crosschaininput1" }

func (cci *CrossChainInput) writeForHash(w io.Writer) {
	mustWriteForHash(w, cci.MainchainOutputId)
}

// SetDestination will link the CrossChainInput to the output
func (cci *CrossChainInput) SetDestination(id *Hash, val *AssetAmount, pos uint64) {
	cci.WitnessDestination = &ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
	}
}

// NewCrossChainInput creates a new CrossChainInput.
func NewCrossChainInput(mainchainOutputID *Hash, ordinal uint64) *CrossChainInput {
	return &CrossChainInput{
		MainchainOutputId: mainchainOutputID,
		Value:             value,
		Ordinal:           ordinal,
	}
}
