package database

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"

	"golang.org/x/crypto/sha3"

	"github.com/vapor/application/mov/common"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/protocol/bc"
)

const (
	order byte = iota
	tradePair
	matchStatus

	tradePairsNum = 1024
	ordersNum     = 10240
	AssetIDLen    = 32
)

var (
	movStore        = []byte("MOV:")
	ordersPreFix    = append(movStore, order)
	tradePairPreFix = append(movStore, tradePair)
	bestMatchStore  = append(movStore, matchStatus)
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
	key := append(tradePairPreFix, fromAssetID.Bytes()...)
	return append(key, toAssetID.Bytes()...)
}

type MovStore struct {
	db dbm.DB
}

func NewMovStore(db dbm.DB) *MovStore {
	return &MovStore{db: db}
}

func (d *MovStore) ListOrders(fromAssetID, toAssetID *bc.AssetID, rateAfter float64) ([]*common.Order, error) {
	if fromAssetID == nil || toAssetID == nil {
		return nil, errors.New("assetID is nil")
	}

	ordersPreFixLen := len(ordersPreFix)
	orderPreFix := append(ordersPreFix, fromAssetID.Bytes()...)
	orderPreFix = append(orderPreFix, toAssetID.Bytes()...)

	var startKey []byte
	if rateAfter > 0 {
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, math.Float64bits(rateAfter))
		copy(startKey, orderPreFix)
		startKey = append(startKey, buf...)
	}

	itr := d.db.IteratorPrefixWithStart(orderPreFix, startKey, false)
	defer itr.Release()

	var orders []*common.Order
	for txNum := ordersNum; itr.Next() && txNum > 0; txNum-- {
		key := itr.Key()
		ratePos := ordersPreFixLen + AssetIDLen*2
		rate := math.Float64frombits(binary.BigEndian.Uint64(key[ratePos : ratePos+8]))

		movUtxo := &common.MovUtxo{}
		if err := json.Unmarshal(itr.Value(), movUtxo); err != nil {
			return nil, err
		}

		order := &common.Order{
			FromAssetID: fromAssetID,
			ToAssetID:   toAssetID,
			Rate:        rate,
			Utxo:        movUtxo,
		}

		orders = append(orders, order)
	}

	return orders, nil
}

func (d *MovStore) ProcessOrders(addOrders []*common.Order, delOreders []*common.Order, height uint64, blockHash *bc.Hash) error {
	batch := d.db.NewBatch()

	if err := d.addOrders(batch, addOrders); err != nil {
		return err
	}

	if err := d.deleteOrder(batch, delOreders); err != nil {
		return err
	}

	if err := d.saveMovDatabaseState(batch, &common.MovDatabaseState{Height: height, Hash: blockHash}); err != nil {
		return err
	}

	batch.Write()
	return nil
}

func (d *MovStore) addOrders(batch dbm.Batch, orders []*common.Order) error {
	tradePairMap := make(map[common.TradePair]uint64)
	for _, order := range orders {
		data, err := json.Marshal(order.Utxo)
		if err != nil {
			return err
		}

		utxoHash := bc.NewHash(sha3.Sum256(data))
		key := calcOrdersKey(order.FromAssetID, order.ToAssetID, &utxoHash, order.Rate)
		batch.Set(key, data)

		tradePair := common.TradePair{
			FromAssetID: order.FromAssetID,
			ToAssetID:   order.ToAssetID,
		}
		tradePairMap[tradePair] += 1
	}

	if err := d.addTradePair(batch, tradePairMap); err != nil {
		return err
	}
	return nil
}

func (d *MovStore) deleteOrder(batch dbm.Batch, orders []*common.Order) error {
	tradePairMap := make(map[common.TradePair]uint64)
	for _, order := range orders {
		data, err := json.Marshal(order.Utxo)
		if err != nil {
			return err
		}

		utxoHash := bc.NewHash(sha3.Sum256(data))
		key := calcOrdersKey(order.FromAssetID, order.ToAssetID, &utxoHash, order.Rate)
		batch.Delete(key)
		tradePair := common.TradePair{
			FromAssetID: order.FromAssetID,
			ToAssetID:   order.ToAssetID,
		}
		tradePairMap[tradePair] += 1
	}

	if err := d.deleteTradePair(batch, tradePairMap); err != nil {
		return err
	}
	return nil
}

func (d *MovStore) GetMovDatabaseState() (*common.MovDatabaseState, error) {
	value := d.db.Get(bestMatchStore)
	if value != nil {
		return nil, errors.New("don't find state of mov-database")
	}

	state := &common.MovDatabaseState{}
	if err := json.Unmarshal(value, state); err != nil {
		return nil, err
	}

	return state, nil
}

func (d *MovStore) ListTradePairsWithStart(fromAssetIDAfter, toAssetIDAfter *bc.AssetID) ([]*common.TradePair, error) {
	var startKey []byte
	if fromAssetIDAfter != nil && toAssetIDAfter != nil {
		startKey = calcTradePairKey(fromAssetIDAfter, toAssetIDAfter)
	}

	preFixLen := len(tradePairPreFix)
	itr := d.db.IteratorPrefixWithStart(tradePairPreFix, startKey, false)
	defer itr.Release()

	var tradePairs []*common.TradePair
	for txNum := tradePairsNum; itr.Next() && txNum > 0; txNum-- {
		key := itr.Key()
		b := [32]byte{}
		fromAssetIDPos := preFixLen
		copy(b[:], key[fromAssetIDPos:fromAssetIDPos+AssetIDLen])
		fromAssetID := bc.NewAssetID(b)

		toAssetIDPos := preFixLen + AssetIDLen
		copy(b[:], key[toAssetIDPos:toAssetIDPos+AssetIDLen])
		toAssetID := bc.NewAssetID(b)

		count := binary.BigEndian.Uint64(itr.Value())

		tradePairs = append(tradePairs, &common.TradePair{FromAssetID: &fromAssetID, ToAssetID: &toAssetID, Count: count})
	}

	return tradePairs, nil
}

func (d *MovStore) addTradePair(batch dbm.Batch, tradePairs map[common.TradePair]uint64) error {
	for k, v := range tradePairs {
		count := uint64(0)
		key := calcTradePairKey(k.FromAssetID, k.ToAssetID)
		if value := d.db.Get(key); value != nil {
			count = binary.BigEndian.Uint64(value)
		}

		count += v

		value := [8]byte{}
		binary.BigEndian.PutUint64(value[:], count)
		batch.Set(key, value[:])
	}
	return nil
}

func (d *MovStore) deleteTradePair(batch dbm.Batch, tradePairs map[common.TradePair]uint64) error {
	for k, v := range tradePairs {
		key := calcTradePairKey(k.FromAssetID, k.ToAssetID)
		value := d.db.Get(key)
		if value == nil {
			return errors.New("don't find trade pair")
		}
		count := binary.BigEndian.Uint64(value) - v

		if count > 0 {
			value := [8]byte{}
			binary.BigEndian.PutUint64(value[:], count)
			batch.Set(key, value[:])
		} else {
			batch.Delete(key)
		}
	}

	return nil
}

func (d *MovStore) saveMovDatabaseState(batch dbm.Batch, state *common.MovDatabaseState) error {
	value, err := json.Marshal(state)
	if err != nil {
		return err
	}

	batch.Set(bestMatchStore, value)
	return nil
}
