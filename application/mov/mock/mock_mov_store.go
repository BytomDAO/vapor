package mock

import (
	"sort"

	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
)

type MovStore struct {
	tradePairs []*common.TradePair
	orderMap   map[string][]*common.Order
	dbState    *common.MovDatabaseState
}

func NewMovStore(tradePairs []*common.TradePair, orders []*common.Order) *MovStore {
	orderMap := make(map[string][]*common.Order)
	for _, order := range orders {
		orderMap[order.TradePair().Key()] = append(orderMap[order.TradePair().Key()], order)
	}

	for _, orders := range orderMap {
		sort.Sort(common.OrderSlice(orders))
	}
	return &MovStore{
		tradePairs: tradePairs,
		orderMap:   orderMap,
	}
}

func (m *MovStore) GetMovDatabaseState() (*common.MovDatabaseState, error) {
	return m.dbState, nil
}

func (m *MovStore) InitDBState(height uint64, hash *bc.Hash) error {
	return nil
}

func (m *MovStore) ListOrders(orderAfter *common.Order) ([]*common.Order, error) {
	tradePair := &common.TradePair{FromAssetID: orderAfter.FromAssetID, ToAssetID: orderAfter.ToAssetID}
	orders := m.orderMap[tradePair.Key()]
	begin := len(orders)
	if orderAfter.Rate() == 0 {
		begin = 0
	} else {
		for i, order := range orders {
			if order.Rate() == orderAfter.Rate() {
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

func (m *MovStore) ListTradePairsWithStart(fromAssetIDAfter, toAssetIDAfter *bc.AssetID) ([]*common.TradePair, error) {
	begin := len(m.tradePairs)
	if fromAssetIDAfter == nil || toAssetIDAfter == nil {
		begin = 0
	} else {
		for i, tradePair := range m.tradePairs {
			if *tradePair.FromAssetID == *fromAssetIDAfter && *tradePair.ToAssetID == *toAssetIDAfter {
				begin = i + 1
				break
			}
		}
	}
	var result []*common.TradePair
	for i := begin; i < len(m.tradePairs) && len(result) < 3; i++ {
		result = append(result, m.tradePairs[i])
	}
	return result, nil
}

func (m *MovStore) ProcessOrders(addOrders []*common.Order, delOrders []*common.Order, blockHeader *types.BlockHeader) error {
	for _, order := range addOrders {
		tradePair := &common.TradePair{FromAssetID: order.FromAssetID, ToAssetID: order.ToAssetID}
		m.orderMap[tradePair.Key()] = append(m.orderMap[tradePair.Key()], order)
	}
	for _, delOrder := range delOrders {
		tradePair := &common.TradePair{FromAssetID: delOrder.FromAssetID, ToAssetID: delOrder.ToAssetID}
		orders := m.orderMap[tradePair.Key()]
		for i, order := range orders {
			if delOrder.Key() == order.Key() {
				m.orderMap[tradePair.Key()] = append(orders[0:i], orders[i+1:]...)
			}
		}
	}
	for _, orders := range m.orderMap {
		sort.Sort(common.OrderSlice(orders))
	}

	if blockHeader.Height == m.dbState.Height {
		m.dbState = &common.MovDatabaseState{Height: blockHeader.Height - 1, Hash: &blockHeader.PreviousBlockHash}
	} else if blockHeader.Height == m.dbState.Height+1 {
		blockHash := blockHeader.Hash()
		m.dbState = &common.MovDatabaseState{Height: blockHeader.Height, Hash: &blockHash}
	} else {
		return errors.New("error block header")
	}
	return nil
}
