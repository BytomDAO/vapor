package database

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"

	"github.com/vapor/application/dex/common"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/protocol/bc"
	"golang.org/x/crypto/sha3"
)

const (
	order byte = iota
	tradePair
	matchStatus

	tradePairsNum = 1024
	ordersNum     = 10240
)

var (
	dexStore        = []byte("DEX:")
	ordersPreFix    = append(dexStore, order)
	tradePairPreFix = append(dexStore, tradePair)
	bestMatchStore  = append(dexStore, matchStatus)
)

func calcOrdersKey(fromAssetID, toAssetID *bc.AssetID, utxoHash *bc.Hash, rate float64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, math.Float64bits(rate))
	key := append(ordersPreFix, fromAssetID.Bytes()...)
	key = append(key, toAssetID.Bytes()...)
	key = append(key, buf...)
	return append(key, utxoHash.Bytes()...)
}

func calcTradePairKey(fromAssetID, toAssetID *bc.AssetID) []byte {
	key := append(ordersPreFix, fromAssetID.Bytes()...)
	return append(key, toAssetID.Bytes()...)
}

type DexTradeOrderDB struct {
	db dbm.DB
}

func (d *DexTradeOrderDB) GetTradePairsWithStart(fromAssetID, toAssetID *bc.AssetID) ([]*common.TradePair, error) {
	startKey := []byte{}
	if fromAssetID != nil && toAssetID != nil {
		startKey = calcTradePairKey(fromAssetID, toAssetID)
	}

	tradePairs := []*common.TradePair{}
	preFixLen := len(tradePairPreFix)
	itr := d.db.IteratorPrefixWithStart(tradePairPreFix, startKey, false)
	defer itr.Release()

	for txNum := tradePairsNum; itr.Next() && txNum > 0; {
		key := itr.Key()
		b := [32]byte{}
		fromAssetIDPos := preFixLen
		copy(b[:], key[fromAssetIDPos:fromAssetIDPos+32])
		fromAssetID := bc.NewAssetID(b)

		toAssetIDPos := preFixLen + 32
		copy(b[:], key[toAssetIDPos:toAssetIDPos+32])
		toAssetID := bc.NewAssetID(b)

		count := binary.BigEndian.Uint64(itr.Value())

		tradePairs = append(tradePairs, &common.TradePair{FromAssetID: &fromAssetID, ToAssetID: &toAssetID, Count: count})

		txNum--
	}

	return tradePairs, nil
}

func (d *DexTradeOrderDB) addTradePair(fromAssetID, toAssetID *bc.AssetID) error {
	count := uint64(0)
	key := calcTradePairKey(fromAssetID, toAssetID)
	if value := d.db.Get(key); value != nil {
		count = binary.BigEndian.Uint64(value)
	}

	count++

	value := [8]byte{}
	binary.BigEndian.PutUint64(value[:], count)
	d.db.Set(key, value[:])
	return nil
}

func (d *DexTradeOrderDB) deleteTradePair(fromAssetID, toAssetID *bc.AssetID) error {
	key := calcTradePairKey(fromAssetID, toAssetID)
	if value := d.db.Get(key); value != nil {
		count := binary.BigEndian.Uint64(value) - 1

		if count > 0 {
			value := [8]byte{}
			binary.BigEndian.PutUint64(value[:], count)
			d.db.Set(key, value[:])
		} else {
			d.db.Delete(key)
		}
		return nil
	}

	return errors.New("don't find trade pair")
}

func (d *DexTradeOrderDB) ProcessOrders(addOrders []*common.Order, delOreders []*common.Order, height uint64, blockHash *bc.Hash) error {
	batch := d.db.NewBatch()

	if err := d.addOrders(batch, addOrders); err != nil {
		return err
	}

	if err := d.deleteOrder(batch, delOreders); err != nil {
		return err
	}

	if err := d.saveDexDatabaseState(batch, &common.DexDatabaseState{Height: height, Hash: blockHash}); err != nil {
		return err
	}

	batch.Write()
	return nil
}

func (d *DexTradeOrderDB) addOrders(batch dbm.Batch, orders []*common.Order) error {
	for _, order := range orders {
		data, err := json.Marshal(order.Utxo)
		if err != nil {
			return err
		}
		utxoHash := bc.NewHash(sha3.Sum256(data))
		key := calcOrdersKey(order.FromAssetID, order.ToAssetID, &utxoHash, order.Rate)
		batch.Set(key, data)

		if err := d.addTradePair(order.FromAssetID, order.ToAssetID); err != nil {
			return err
		}
	}
	return nil
}

func (d *DexTradeOrderDB) deleteOrder(batch dbm.Batch, orders []*common.Order) error {
	for _, order := range orders {
		data, err := json.Marshal(order.Utxo)
		if err != nil {
			return err
		}
		utxoHash := bc.NewHash(sha3.Sum256(data))
		key := calcOrdersKey(order.FromAssetID, order.ToAssetID, &utxoHash, order.Rate)
		batch.Delete(key)

		if err := d.deleteTradePair(order.FromAssetID, order.ToAssetID); err != nil {
			return err
		}
	}
	return nil
}

func (d *DexTradeOrderDB) ListOrders(fromAssetID, toAssetID *bc.AssetID, rateAfter float64) ([]*common.Order, error) {
	ordersPreFixLen := len(ordersPreFix)
	orders := []*common.Order{}

	orderPreFix := append(ordersPreFix, fromAssetID.Bytes()...)
	orderPreFix = append(orderPreFix, toAssetID.Bytes()...)

	startKey := []byte{}
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, math.Float64bits(rateAfter))
	copy(startKey, orderPreFix)
	startKey = append(startKey, buf...)

	itr := d.db.IteratorPrefixWithStart(orderPreFix, startKey, false)
	defer itr.Release()

	for txNum := ordersNum; itr.Next() && txNum > 0; {
		key := itr.Key()
		b := [32]byte{}
		fromAssetIDPos := ordersPreFixLen
		copy(b[:], key[fromAssetIDPos:fromAssetIDPos+32])
		fromAssetID := bc.NewAssetID(b)

		toAssetIDPos := ordersPreFixLen + 32
		copy(b[:], key[toAssetIDPos:toAssetIDPos+32])
		toAssetID := bc.NewAssetID(b)

		ratePos := ordersPreFixLen + 32*2
		rate := math.Float64frombits(binary.BigEndian.Uint64(key[ratePos : ratePos+8]))

		dexUtxo := &common.DexUtxo{}
		if err := json.Unmarshal(itr.Value(), dexUtxo); err != nil {
			return nil, err
		}

		order := &common.Order{
			FromAssetID: &fromAssetID,
			ToAssetID:   &toAssetID,
			Rate:        rate,
			Utxo:        dexUtxo,
		}

		orders = append(orders, order)
		txNum--
	}

	return orders, nil
}

func (d *DexTradeOrderDB) GetDexDatabaseState() (*common.DexDatabaseState, error) {
	value := d.db.Get(bestMatchStore)
	if value != nil {
		return nil, errors.New("don't find state of dex-database")
	}

	state := &common.DexDatabaseState{}
	if err := json.Unmarshal(value, state); err != nil {
		return nil, err
	}

	return state, nil
}

func (d *DexTradeOrderDB) saveDexDatabaseState(batch dbm.Batch, state *common.DexDatabaseState) error {
	value, err := json.Marshal(state)
	if err != nil {
		return err
	}

	d.db.Set(bestMatchStore, value)
	return nil
}
