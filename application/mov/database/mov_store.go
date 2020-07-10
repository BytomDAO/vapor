package database

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"

	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/consensus/segwit"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
)

// ErrNotInitDBState represent the database state of mov store is not initialized
var ErrNotInitDBState = errors.New("database state of mov store is not initialized")

// MovStore is the interface for mov's persistent storage
type MovStore interface {
	Clear()
	GetMovDatabaseState() (*common.MovDatabaseState, error)
	InitDBState(height uint64, hash *bc.Hash) error
	ListOrders(orderAfter *common.Order) ([]*common.Order, error)
	ListTradePairsWithStart(fromAssetIDAfter, toAssetIDAfter *bc.AssetID) ([]*common.TradePair, error)
	ProcessOrders(addOrders []*common.Order, delOrders []*common.Order, blockHeader *types.BlockHeader) error
}

const (
	order byte = iota + 1
	tradePair
	matchStatus

	fromAssetIDPos = 0
	toAssetIDPos   = 1
	assetIDLen     = 32
	rateByteLen    = 8

	tradePairsNum = 32
	ordersNum     = 128
)

var (
	movStore         = []byte("MOV:")
	ordersPrefix     = append(movStore, order)
	tradePairsPrefix = append(movStore, tradePair)
	bestMatchStore   = append(movStore, matchStatus)
)

type orderData struct {
	Utxo             *common.MovUtxo
	RatioNumerator   int64
	RatioDenominator int64
	BlockHeight      uint64
	TxIndex          int
}

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

func getAssetIDFromTradePairKey(key []byte, posIndex int) *bc.AssetID {
	b := [32]byte{}
	pos := len(tradePairsPrefix) + assetIDLen*posIndex
	copy(b[:], key[pos:pos+assetIDLen])
	assetID := bc.NewAssetID(b)
	return &assetID
}

func getRateFromOrderKey(key []byte) float64 {
	ratePos := len(ordersPrefix) + assetIDLen*2
	return math.Float64frombits(binary.BigEndian.Uint64(key[ratePos : ratePos+rateByteLen]))
}

type tradePairData struct {
	Count int
}

// LevelDBMovStore is the LevelDB implementation for MovStore
type LevelDBMovStore struct {
	db dbm.DB
}

// NewLevelDBMovStore create a new LevelDBMovStore object
func NewLevelDBMovStore(db dbm.DB) *LevelDBMovStore {
	return &LevelDBMovStore{db: db}
}

// Clear will clear all the data of store
func (m *LevelDBMovStore) Clear() {
	batch := m.db.NewBatch()

	iter := m.db.Iterator()
	defer iter.Release()

	for iter.Next() {
		batch.Delete(iter.Key())
	}
	batch.Write()
}

// GetMovDatabaseState return the current DB's image status
func (m *LevelDBMovStore) GetMovDatabaseState() (*common.MovDatabaseState, error) {
	if value := m.db.Get(bestMatchStore); value != nil {
		state := &common.MovDatabaseState{}
		return state, json.Unmarshal(value, state)
	}

	return nil, ErrNotInitDBState
}

// InitDBState set the DB's image status
func (m *LevelDBMovStore) InitDBState(height uint64, hash *bc.Hash) error {
	state := &common.MovDatabaseState{Height: height, Hash: hash}
	value, err := json.Marshal(state)
	if err != nil {
		return err
	}

	m.db.Set(bestMatchStore, value)
	return nil
}

// ListOrders return n orders after the input order
func (m *LevelDBMovStore) ListOrders(orderAfter *common.Order) ([]*common.Order, error) {
	if orderAfter.FromAssetID == nil || orderAfter.ToAssetID == nil {
		return nil, errors.New("assetID is nil")
	}

	orderPrefix := append(ordersPrefix, orderAfter.FromAssetID.Bytes()...)
	orderPrefix = append(orderPrefix, orderAfter.ToAssetID.Bytes()...)

	var startKey []byte
	if orderAfter.Rate() > 0 {
		startKey = calcOrderKey(orderAfter.FromAssetID, orderAfter.ToAssetID, orderAfter.UTXOHash(), orderAfter.Rate())
	}

	itr := m.db.IteratorPrefixWithStart(orderPrefix, startKey, false)
	defer itr.Release()

	var orders []*common.Order
	for txNum := 0; txNum < ordersNum && itr.Next(); txNum++ {
		orderData := &orderData{}
		if err := json.Unmarshal(itr.Value(), orderData); err != nil {
			return nil, err
		}

		contractArgs, err := segwit.DecodeP2WMCProgram(orderData.Utxo.ControlProgram)
		if err != nil {
			return nil, err
		}

		orders = append(orders, &common.Order{
			FromAssetID:      orderAfter.FromAssetID,
			ToAssetID:        orderAfter.ToAssetID,
			Utxo:             orderData.Utxo,
			RatioNumerator:   orderData.RatioNumerator,
			RatioDenominator: orderData.RatioDenominator,
			BlockHeight:      orderData.BlockHeight,
			TxIndex:          orderData.TxIndex,
			ContractArgs:     contractArgs,
		})
	}
	return orders, nil
}

// ListTradePairsWithStart return n trade pairs after the input trade pair
func (m *LevelDBMovStore) ListTradePairsWithStart(fromAssetIDAfter, toAssetIDAfter *bc.AssetID) ([]*common.TradePair, error) {
	var startKey []byte
	if fromAssetIDAfter != nil && toAssetIDAfter != nil {
		startKey = calcTradePairKey(fromAssetIDAfter, toAssetIDAfter)
	}

	itr := m.db.IteratorPrefixWithStart(tradePairsPrefix, startKey, false)
	defer itr.Release()

	var tradePairs []*common.TradePair
	for txNum := 0; txNum < tradePairsNum && itr.Next(); txNum++ {
		key := itr.Key()
		fromAssetID := getAssetIDFromTradePairKey(key, fromAssetIDPos)
		toAssetID := getAssetIDFromTradePairKey(key, toAssetIDPos)

		tradePairData := &tradePairData{}
		if err := json.Unmarshal(itr.Value(), tradePairData); err != nil {
			return nil, err
		}

		tradePairs = append(tradePairs, &common.TradePair{FromAssetID: fromAssetID, ToAssetID: toAssetID, Count: tradePairData.Count})
	}
	return tradePairs, nil
}

// ProcessOrders update the DB's image by add new orders, delete the used order
func (m *LevelDBMovStore) ProcessOrders(addOrders []*common.Order, delOrders []*common.Order, blockHeader *types.BlockHeader) error {
	if err := m.checkMovDatabaseState(blockHeader); err != nil {
		return err
	}

	batch := m.db.NewBatch()
	tradePairsCnt := make(map[string]*common.TradePair)
	if err := m.addOrders(batch, addOrders, tradePairsCnt); err != nil {
		return err
	}

	m.deleteOrders(batch, delOrders, tradePairsCnt)
	if err := m.updateTradePairs(batch, tradePairsCnt); err != nil {
		return err
	}

	state, err := m.calcNextDatabaseState(blockHeader)
	if err != nil {
		return err
	}

	if err := m.saveMovDatabaseState(batch, state); err != nil {
		return err
	}

	batch.Write()
	return nil
}

func (m *LevelDBMovStore) addOrders(batch dbm.Batch, orders []*common.Order, tradePairsCnt map[string]*common.TradePair) error {
	for _, order := range orders {
		orderData := &orderData{
			Utxo:             order.Utxo,
			RatioNumerator:   order.RatioNumerator,
			RatioDenominator: order.RatioDenominator,
			BlockHeight:      order.BlockHeight,
			TxIndex:          order.TxIndex,
		}
		data, err := json.Marshal(orderData)
		if err != nil {
			return err
		}

		key := calcOrderKey(order.FromAssetID, order.ToAssetID, order.UTXOHash(), order.Rate())
		batch.Set(key, data)

		tradePair := &common.TradePair{
			FromAssetID: order.FromAssetID,
			ToAssetID:   order.ToAssetID,
		}
		if _, ok := tradePairsCnt[tradePair.Key()]; !ok {
			tradePairsCnt[tradePair.Key()] = tradePair
		}
		tradePairsCnt[tradePair.Key()].Count++
	}
	return nil
}

func (m *LevelDBMovStore) calcNextDatabaseState(blockHeader *types.BlockHeader) (*common.MovDatabaseState, error) {
	hash := blockHeader.Hash()
	height := blockHeader.Height

	state, err := m.GetMovDatabaseState()
	if err != nil {
		return nil, err
	}

	if *state.Hash == hash {
		hash = blockHeader.PreviousBlockHash
		height = blockHeader.Height - 1
	}

	return &common.MovDatabaseState{Height: height, Hash: &hash}, nil
}

func (m *LevelDBMovStore) checkMovDatabaseState(header *types.BlockHeader) error {
	state, err := m.GetMovDatabaseState()
	if err != nil {
		return err
	}

	if (*state.Hash == header.PreviousBlockHash && (state.Height+1) == header.Height) || *state.Hash == header.Hash() {
		return nil
	}

	return errors.New("the status of the block is inconsistent with that of mov-database")
}

func (m *LevelDBMovStore) deleteOrders(batch dbm.Batch, orders []*common.Order, tradePairsCnt map[string]*common.TradePair) {
	for _, order := range orders {
		key := calcOrderKey(order.FromAssetID, order.ToAssetID, order.UTXOHash(), order.Rate())
		batch.Delete(key)

		tradePair := &common.TradePair{
			FromAssetID: order.FromAssetID,
			ToAssetID:   order.ToAssetID,
		}
		if _, ok := tradePairsCnt[tradePair.Key()]; !ok {
			tradePairsCnt[tradePair.Key()] = tradePair
		}
		tradePairsCnt[tradePair.Key()].Count--
	}
}

func (m *LevelDBMovStore) saveMovDatabaseState(batch dbm.Batch, state *common.MovDatabaseState) error {
	value, err := json.Marshal(state)
	if err != nil {
		return err
	}

	batch.Set(bestMatchStore, value)
	return nil
}

func (m *LevelDBMovStore) updateTradePairs(batch dbm.Batch, tradePairs map[string]*common.TradePair) error {
	for _, v := range tradePairs {
		key := calcTradePairKey(v.FromAssetID, v.ToAssetID)
		tradePairData := &tradePairData{}
		if value := m.db.Get(key); value != nil {
			if err := json.Unmarshal(value, tradePairData); err != nil {
				return err
			}
		}

		if tradePairData.Count += v.Count; tradePairData.Count < 0 {
			return errors.New("negative trade count")
		}

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
