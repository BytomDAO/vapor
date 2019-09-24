package database

import (
	"encoding/binary"
	"math"

	"github.com/vapor/application/dex/common"
	dbm "github.com/vapor/database/leveldb"
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

func (d *DexTradeOrderDB) GetTradePairsWithStart(start []byte) (error, []common.TradePair) {
	return nil, nil
}

func (d *DexTradeOrderDB) addTradePair() error {
	return nil
}

func (d *DexTradeOrderDB) deleteTradePair() error {
	return nil
}

func (d *DexTradeOrderDB) ProcessOrders(orders []*common.Order, delOreders []*common.Order, height uint64, blockHash *bc.Hash) error {

	return nil
}

func (d *DexTradeOrderDB) addOrders(orders []*common.Order) error {
	return nil
}

func (d *DexTradeOrderDB) deleteOrder(orders []*common.Order) error {
	return nil
}

func (d *DexTradeOrderDB) ListOrders(fromAssetID, toAssetID string, rateAfter float64) (error, []*common.Order) {
	return nil, nil
}

func (d *DexTradeOrderDB) GetDexDatabaseState() (error, *common.DexDatabaseState) {
	return nil, nil
}

func (d *DexTradeOrderDB) saveDexDatabaseState(state *common.DexDatabaseState) error {
	return nil
}
