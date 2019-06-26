package orm

import (
	"encoding/json"

	"github.com/vapor/federation/types"
)

type CrossTransactionReq struct {
	ID                 uint64 `gorm:"primary_key"`
	CrossTransactionID uint64
	SourcePos          uint64
	AssetID            uint64
	AssetAmount        uint64
	Script             string
	CreatedAt          types.Timestamp
	UpdatedAt          types.Timestamp

	CrossTransaction *CrossTransaction `gorm:"foreignkey:CrossTransactionID"`
	Asset            *Asset            `gorm:"foreignkey:ID"`
}

func (c *CrossTransactionReq) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		AssetID   string `json:"asset_id"`
		Amount    uint64 `json:"amount"`
		ToAddress string `json:"to_address"`
	}{
		Amount:    c.AssetAmount,
		ToAddress: ",",
		AssetID:   c.Asset.AssetID,
	})
}
