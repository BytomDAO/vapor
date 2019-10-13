package common

import (
	"fmt"

	"github.com/vapor/protocol/bc"
)

type MovUtxo struct {
	SourceID       *bc.Hash
	SourcePos      uint64
	Amount         uint64
	ControlProgram []byte
}

type Order struct {
	FromAssetID *bc.AssetID
	ToAssetID   *bc.AssetID
	Utxo        *MovUtxo
	Rate        float64
}

func (o *Order) ID() string {
	return fmt.Sprintf("%s:%d", o.Utxo.SourceID, o.Utxo.SourcePos)
}

type OrderSlice []*Order

func (o OrderSlice) Len() int {
	return len(o)
}
func (o OrderSlice) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}
func (o OrderSlice) Less(i, j int) bool {
	return o[i].Rate < o[j].Rate
}

type TradePair struct {
	FromAssetID *bc.AssetID
	ToAssetID   *bc.AssetID
	Count       int
}

func (t *TradePair) Reverse() *TradePair {
	return &TradePair{
		FromAssetID: t.ToAssetID,
		ToAssetID:   t.FromAssetID,
	}
}

func (t *TradePair) ID() string {
	return fmt.Sprintf("%s:%s", t.FromAssetID, t.ToAssetID)
}

type MovDatabaseState struct {
	Height uint64
	Hash   *bc.Hash
}
