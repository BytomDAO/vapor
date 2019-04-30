package orm

import "github.com/vapor/protocol/bc/types"

type Transaction struct {
	BlockHeaderID  uint
	BlockHash      string
	BlockHeight    uint64
	Version        uint64
	BlockTimestamp uint64
	TxIndex        uint64
	RawData        string
	StatusFail     bool
}

func (t *Transaction) UnmarshalText() (*types.Tx, error) {
	tx := &types.Tx{}
	if err := tx.UnmarshalText([]byte(t.RawData)); err != nil {
		return nil, err
	}
	return tx, nil
}
