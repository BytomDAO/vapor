package orm

type BlockStoreState struct {
	StoreKey string `gorm:"primary_key"`
	Height   uint64
	Hash     string
}

type BlockHeader struct {
	Height                 uint64 `gorm:"primary_key"`
	BlockHash              string `gorm:"primary_key"`
	Version                uint64
	PreviousBlockHash      string
	Timestamp              uint64
	TransactionsMerkleRoot string
	TransactionStatusHash  string
}

type Block struct {
	BlockHash string
	Height    uint64
	Block     string
	Header    string
	TxStatus  string
}
