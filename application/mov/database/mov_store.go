package database

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"

	"github.com/bytom/vapor/application/mov/common"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
)

const (
	order byte = iota
	tradePair
	matchStatus

	tradePairsNum = 1024
	ordersNum     = 10240
	assetIDLen    = 32
	rateByteLen   = 8
)

var (
	movStore         = []byte("MOV:")
	ordersPrefix     = append(movStore, order)
	tradePairsPrefix = append(movStore, tradePair)
	bestMatchStore   = append(movStore, matchStatus)
)

func calcOrderKey(fromAssetID, toAssetID *bc.AssetID, utxoHash *bc.Hash, rate float64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, math.Float64bits(rate))
	key := append(ordersPrefix, fromAssetID.Bytes()...)
	key = append(key, toAssetID.Bytes()...)
	key = append(key, buf...)
	return append(key, utxoHash.Bytes()...)
}

func calcTradePairKey(fromAssetID, toAssetID *bc.AssetID) []byte {
	key := append(tradePairsPrefix, fromAssetID.Bytes()...)
	return append(key, toAssetID.Bytes()...)
}

func calcUTXOHash(order *common.Order) *bc.Hash {
	prog := &bc.Program{VmVersion: 1, Code: order.Utxo.ControlProgram}
	src := &bc.ValueSource{
		Ref:      order.Utxo.SourceID,
		Value:    &bc.AssetAmount{AssetId: order.FromAssetID, Amount: order.Utxo.Amount},
		Position: order.Utxo.SourcePos,
	}
	hash := bc.EntryID(bc.NewIntraChainOutput(src, prog, 0))
	return &hash
}

func getAssetIDFromTradePairKey(key []byte, prefix []byte, posIndex int) *bc.AssetID {
	b := [32]byte{}
	pos := len(prefix) + assetIDLen*posIndex
	copy(b[:], key[pos:pos+assetIDLen])
	assetID := bc.NewAssetID(b)
	return &assetID
}

func getRateFromOrderKey(key []byte, prefix []byte) float64 {
	ratePos := len(prefix) + assetIDLen*2
	return math.Float64frombits(binary.BigEndian.Uint64(key[ratePos : ratePos+rateByteLen]))
}

type tradePairData struct {
	Count int
}

type MovStore struct {
	db dbm.DB
}

func NewMovStore(db dbm.DB, height uint64, hash *bc.Hash) (*MovStore, error) {
	if value := db.Get(bestMatchStore); value == nil {
		state := &common.MovDatabaseState{Height: height, Hash: hash}
		value, err := json.Marshal(state)
		if err != nil {
			return nil, err
		}

		db.Set(bestMatchStore, value)
	}
	return &MovStore{db: db}, nil
}

func (m *MovStore) ListOrders(orderAfter *common.Order) ([]*common.Order, error) {
	if orderAfter.FromAssetID == nil || orderAfter.ToAssetID == nil {
		return nil, errors.New("assetID is nil")
	}

	orderPrefix := append(ordersPrefix, orderAfter.FromAssetID.Bytes()...)
	orderPrefix = append(orderPrefix, orderAfter.ToAssetID.Bytes()...)

	var startKey []byte
	if orderAfter.Rate > 0 {
		startKey = calcOrderKey(orderAfter.FromAssetID, orderAfter.ToAssetID, calcUTXOHash(orderAfter), orderAfter.Rate)
	}

	itr := m.db.IteratorPrefixWithStart(orderPrefix, startKey, false)
	defer itr.Release()

	var orders []*common.Order
	for txNum := 0; txNum < ordersNum && itr.Next(); txNum++ {
		movUtxo := &common.MovUtxo{}
		if err := json.Unmarshal(itr.Value(), movUtxo); err != nil {
			return nil, err
		}

		order := &common.Order{
			FromAssetID: orderAfter.FromAssetID,
			ToAssetID:   orderAfter.ToAssetID,
			Rate:        getRateFromOrderKey(itr.Key(), ordersPrefix),
			Utxo:        movUtxo,
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (m *MovStore) ProcessOrders(addOrders []*common.Order, delOreders []*common.Order, blockHeader *types.BlockHeader) error {
	if err := m.checkMovDatabaseState(blockHeader); err != nil {
		return err
	}

	batch := m.db.NewBatch()
	tradePairsCnt := make(map[common.TradePair]int)
	if err := m.addOrders(batch, addOrders, tradePairsCnt); err != nil {
		return err
	}

	m.deleteOrders(batch, delOreders, tradePairsCnt)

	if err := m.updateTradePairs(batch, tradePairsCnt); err != nil {
		return err
	}

	hash := blockHeader.Hash()
	if err := m.saveMovDatabaseState(batch, &common.MovDatabaseState{Height: blockHeader.Height, Hash: &hash}); err != nil {
		return err
	}

	batch.Write()
	return nil
}

func (m *MovStore) addOrders(batch dbm.Batch, orders []*common.Order, tradePairsCnt map[common.TradePair]int) error {
	for _, order := range orders {
		data, err := json.Marshal(order.Utxo)
		if err != nil {
			return err
		}

		key := calcOrderKey(order.FromAssetID, order.ToAssetID, calcUTXOHash(order), order.Rate)
		batch.Set(key, data)

		tradePair := common.TradePair{
			FromAssetID: order.FromAssetID,
			ToAssetID:   order.ToAssetID,
		}
		tradePairsCnt[tradePair] += 1
	}
	return nil
}

func (m *MovStore) deleteOrders(batch dbm.Batch, orders []*common.Order, tradePairsCnt map[common.TradePair]int) {
	for _, order := range orders {
		key := calcOrderKey(order.FromAssetID, order.ToAssetID, calcUTXOHash(order), order.Rate)
		batch.Delete(key)

		tradePair := common.TradePair{
			FromAssetID: order.FromAssetID,
			ToAssetID:   order.ToAssetID,
		}
		tradePairsCnt[tradePair] -= 1
	}
}

func (m *MovStore) GetMovDatabaseState() (*common.MovDatabaseState, error) {
	if value := m.db.Get(bestMatchStore); value != nil {
		state := &common.MovDatabaseState{}
		return state, json.Unmarshal(value, state)
	}

	return nil, errors.New("don't find state of mov-database")
}

func (m *MovStore) ListTradePairsWithStart(fromAssetIDAfter, toAssetIDAfter *bc.AssetID) ([]*common.TradePair, error) {
	var startKey []byte
	if fromAssetIDAfter != nil && toAssetIDAfter != nil {
		startKey = calcTradePairKey(fromAssetIDAfter, toAssetIDAfter)
	}

	itr := m.db.IteratorPrefixWithStart(tradePairsPrefix, startKey, false)
	defer itr.Release()

	var tradePairs []*common.TradePair
	for txNum := 0; txNum < tradePairsNum && itr.Next(); txNum++ {
		key := itr.Key()
		fromAssetID := getAssetIDFromTradePairKey(key, tradePairsPrefix, 0)
		toAssetID := getAssetIDFromTradePairKey(key, tradePairsPrefix, 1)

		tradePairData := &tradePairData{}
		if err := json.Unmarshal(itr.Value(), tradePairData); err != nil {
			return nil, err
		}

		tradePairs = append(tradePairs, &common.TradePair{FromAssetID: fromAssetID, ToAssetID: toAssetID, Count: tradePairData.Count})
	}
	return tradePairs, nil
}

func (m *MovStore) updateTradePairs(batch dbm.Batch, tradePairs map[common.TradePair]int) error {
	for k, v := range tradePairs {
		key := calcTradePairKey(k.FromAssetID, k.ToAssetID)
		tradePairData := &tradePairData{}
		if value := m.db.Get(key); value != nil {
			if err := json.Unmarshal(value, tradePairData); err != nil {
				return err
			}
		} else if v < 0 {
			return errors.New("don't find trade pair")
		}

		tradePairData.Count += v
		if tradePairData.Count > 0 {
			value, err := json.Marshal(tradePairData)
			if err != nil {
				return err
			}

			batch.Set(key, value)
		} else {
			batch.Delete(key)
		}
	}
	return nil
}

func (m *MovStore) checkMovDatabaseState(header *types.BlockHeader) error {
	state, err := m.GetMovDatabaseState()
	if err != nil {
		return err
	}

	blockHash := header.Hash()
	if (state.Hash.String() == header.PreviousBlockHash.String() && (state.Height+1) == header.Height) || state.Hash.String() == blockHash.String() {
		return nil
	}

	return errors.New("the status of the block is inconsistent with that of mov-database")
}

func (m *MovStore) saveMovDatabaseState(batch dbm.Batch, state *common.MovDatabaseState) error {
	value, err := json.Marshal(state)
	if err != nil {
		return err
	}

	batch.Set(bestMatchStore, value)
	return nil
}
