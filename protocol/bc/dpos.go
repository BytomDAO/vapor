package bc

import (
	"io"
)

func (Dpos) typ() string { return "dpos1" }
func (d *Dpos) writeForHash(w io.Writer) {
	mustWriteForHash(w, d.SpentOutputId)
	mustWriteForHash(w, d.Type)
	mustWriteForHash(w, d.From)
	mustWriteForHash(w, d.To)
	mustWriteForHash(w, d.Stake)
}

// SetDestination will link the spend to the output
func (d *Dpos) SetDestination(id *Hash, val *AssetAmount, pos uint64) {
	d.WitnessDestination = &ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
	}
}

// NewDpos creates a new Spend.
func NewDpos(spentOutputID *Hash, ordinal uint64, t uint32, stake uint64, from, to, data string) *Dpos {
	return &Dpos{
		SpentOutputId: spentOutputID,
		Ordinal:       ordinal,
		Type:          t,
		From:          from,
		To:            to,
		Stake:         stake,
		Data:          data,
	}
}
