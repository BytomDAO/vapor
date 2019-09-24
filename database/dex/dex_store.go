package dex

import (
	"encoding/binary"
	"math"

	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/dex"
	"github.com/vapor/protocol/bc"
)

const (
	order byte = iota
	tradePair
	matchStatus
)

var (
	dexStore        = []byte("DEX:")
	OrdersPreFix    = append(dexStore, order)
	TradePairPreFix = append(dexStore, tradePair)
	bestMatchStore  = append(dexStore, matchStatus)
)

func calcOrdersPrefix(fromAssetID, toAssetID *bc.AssetID, utxoHash *bc.Hash, rate float64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, math.Float64bits(rate))
	key := append(OrdersPreFix, fromAssetID.Bytes()...)
	key = append(key, toAssetID.Bytes()...)
	key = append(key, buf...)
	return append(key, utxoHash.Bytes()...)
}

func calcTradePairPreFix(fromAssetID, toAssetID *bc.Hash) []byte {
	key := append(OrdersPreFix, fromAssetID.Bytes()...)
	return append(key, toAssetID.Bytes()...)
}

type DexTradeOrderDB struct {
	db dbm.DB
}

func (d *DexTradeOrderDB) GetTradePairsWithStart(start []byte) []dex.TradePair {
	return nil
}

func (d *DexTradeOrderDB) addTradePair() {

}

func (d *DexTradeOrderDB) deleteTradePair() {

}

func (d *DexTradeOrderDB) ProcessOrders(orders []dex.Order, delOreders []dex.Order, height uint64, blockHash *bc.Hash) {

}

func (d *DexTradeOrderDB) addOrders(orders []dex.Order) {

}

func (d *DexTradeOrderDB) deleteOrder(orders []dex.Order) {

}

func (d *DexTradeOrderDB) ListOrders(fromAssetID, toAssetID string, rateAfter float64) []dex.Order {
	return nil
}

func (d *DexTradeOrderDB) GetMatchState() *dex.MatchState {
	return nil
}

func (d *DexTradeOrderDB) SaveMatchState(state *dex.MatchState) {
}
