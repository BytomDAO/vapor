package bc

import "io"

// CrossChainOutput is the result of a transfer of value. The value it contains
// can never be accessed on side chain, as it has been transfered back to the
// main chain.
// CrossChainOutput satisfies the Entry interface.
// (Not to be confused with the deprecated type TxOutput.)

func (CrossChainOutput) typ() string { return "crosschainoutput1" }
func (o *CrossChainOutput) writeForHash(w io.Writer) {
	mustWriteForHash(w, o.Source)
	mustWriteForHash(w, o.ControlProgram)
}

// NewCrossChainOutput creates a new CrossChainOutput.
func NewCrossChainOutput(source *ValueSource, controlProgram *Program, ordinal uint64) *CrossChainOutput {
	return &CrossChainOutput{
		Source:         source,
		ControlProgram: controlProgram,
		Ordinal:        ordinal,
	}
}
