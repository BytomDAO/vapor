package orm

type Utxo struct {
	ID           uint64 `gorm:"primary_key"`
	Xpub         string
	VoterAddress string
	VoteHeight   uint64
	VoteNum      uint64
	VetoHeight   uint64
	OutputID     string
}
