package orm

type BlockState struct {
	Height    uint64 `json:"height"`
	BlockHash string `json:"block_hash"`
}
