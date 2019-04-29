package orm

type Utxo struct {
	OutputID    string
	IsCoinBase  bool
	BlockHeight uint64
	Spent       bool
}
