package bc

import "io"

// IntraChainOutput is the result of a transfer of value. The value it contains
// may be accessed by a later Spend entry (if that entry can satisfy
// the Output's ControlProgram). IntraChainOutput satisfies the Entry interface.
//
// (Not to be confused with the deprecated type TxOutput.)

func (IntraChainOutput) typ() string { return "intrachainoutput1" }
func (o *IntraChainOutput) writeForHash(w io.Writer) {
	mustWriteForHash(w, o.Source)
	mustWriteForHash(w, o.ControlProgram)
}

// NewIntraChainOutput creates a new IntraChainOutput.
func NewIntraChainOutput(source *ValueSource, controlProgram *Program, ordinal uint64) *IntraChainOutput {
	return &IntraChainOutput{
		Source:         source,
		ControlProgram: controlProgram,
		Ordinal:        ordinal,
	}
}
