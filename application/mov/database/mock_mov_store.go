package database

import (
	"sort"

	"github.com/vapor/application/mov/common"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

type MockMovStore struct {
	TradePairs []*common.TradePair
	OrderMap   map[string][]*common.Order
	DBState    *common.MovDatabaseState
}

func (m *MockMovStore) GetMovDatabaseState() (*common.MovDatabaseState, error) {
	return m.DBState, nil
}

func (m *MockMovStore) InitDBState(height uint64, hash *bc.Hash) error {
	return nil
}

func (m *MockMovStore) ListOrders(orderAfter *common.Order) ([]*common.Order, error) {
	tradePair := &common.TradePair{FromAssetID: orderAfter.FromAssetID, ToAssetID: orderAfter.ToAssetID}
	orders := m.OrderMap[tradePair.Key()]
	begin := len(orders)
	if orderAfter.Rate == 0 {
		begin = 0
	} else {
		for i, order := range orders {
			if order.Rate == orderAfter.Rate {
				begin = i + 1
				break
			}
		}
	}
	var result []*common.Order
	for i := begin; i < len(orders) && len(result) < 3; i++ {
		result = append(result, orders[i])
	}
	return result, nil
}

func (m *MockMovStore) ListTradePairsWithStart(fromAssetIDAfter, toAssetIDAfter *bc.AssetID) ([]*common.TradePair, error) {
	begin := len(m.TradePairs)
	if fromAssetIDAfter == nil || toAssetIDAfter == nil {
		begin = 0
	} else {
		for i, tradePair := range m.TradePairs {
			if *tradePair.FromAssetID == *fromAssetIDAfter && *tradePair.ToAssetID == *toAssetIDAfter {
				begin = i + 1
				break
			}
		}
	}
	var result []*common.TradePair
	for i := begin; i < len(m.TradePairs) && len(result) < 3; i++ {
		result = append(result, m.TradePairs[i])
	}
	return result, nil
}

func (m *MockMovStore) ProcessOrders(addOrders []*common.Order, delOrders []*common.Order, blockHeader *types.BlockHeader) error {
	for _, order := range addOrders {
		tradePair := &common.TradePair{FromAssetID: order.FromAssetID, ToAssetID: order.ToAssetID}
		m.OrderMap[tradePair.Key()] = append(m.OrderMap[tradePair.Key()], order)
	}
	for _, delOrder := range delOrders {
		tradePair := &common.TradePair{FromAssetID: delOrder.FromAssetID, ToAssetID: delOrder.ToAssetID}
		orders := m.OrderMap[tradePair.Key()]
		for i, order := range orders {
			if delOrder.Key() == order.Key() {
				m.OrderMap[tradePair.Key()] = append(orders[0:i], orders[i+1:]...)
			}
		}
	}
	for _, orders := range m.OrderMap {
		sort.Sort(common.OrderSlice(orders))
	}

	if blockHeader.Height == m.DBState.Height {
		m.DBState = &common.MovDatabaseState{Height: blockHeader.Height - 1, Hash: &blockHeader.PreviousBlockHash}
	} else if blockHeader.Height == m.DBState.Height+1 {
		blockHash := blockHeader.Hash()
		m.DBState = &common.MovDatabaseState{Height: blockHeader.Height, Hash: &blockHash}
	} else {
		return errors.New("error block header")
	}
	return nil
}
