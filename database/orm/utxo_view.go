package orm

type UtxoViewpoint struct {
	OutputID    string `gorm:"primary_key"`
	IsCoinBase  bool
	BlockHeight uint64
	Spent       bool
	IsCliam     bool
}
