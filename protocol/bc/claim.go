package bc

import "io"

func (Claim) typ() string { return "claim1" }
func (c *Claim) writeForHash(w io.Writer) {
	mustWriteForHash(w, c.Peginwitness)
}

// SetDestination is support function for map tx
func (c *Claim) SetDestination(id *Hash, val *AssetAmount, pos uint64) {
	c.WitnessDestination = &ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
	}
}

func NewClaim(spentOutputID *Hash, ordinal uint64, peginwitness [][]byte) *Claim {
	return &Claim{
		SpentOutputId: spentOutputID,
		Ordinal:       ordinal,
		Peginwitness:  peginwitness,
	}
}
