package orm

import "github.com/vapor/protocol/bc/types"

type Transaction struct {
	BlockHeaderID uint
	TxIndex       uint64
	RawData       string
	StatusFail    bool

	BlockHeader *BlockHeader `gorm:"FOREIGNKEY:BlockHeaderID;AssociationForeignKey:ID"`
}

func (t *Transaction) UnmarshalText() (*types.Tx, error) {
	tx := &types.Tx{}
	if err := tx.UnmarshalText([]byte(t.RawData)); err != nil {
		return nil, err
	}
	return tx, nil
}
