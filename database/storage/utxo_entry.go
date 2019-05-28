package storage

// NewUtxoEntry will create a new utxo entry
func NewUtxoEntry(isCoinBase bool, blockHeight uint64, spent bool, fromMainchain bool) *UtxoEntry {
	return &UtxoEntry{
		IsCoinBase:    isCoinBase,
		BlockHeight:   blockHeight,
		Spent:         spent,
		FromMainchain: fromMainchain,
	}
}

// SpendOutput marks the output at the provided index as spent
func (entry *UtxoEntry) SpendOutput() {
	entry.Spent = true
}

// UnspendOutput marks the output at the provided index as unspent
func (entry *UtxoEntry) UnspendOutput() {
	entry.Spent = false
}
