package database

import (
	"github.com/vapor/application/mov/common"
	"github.com/vapor/protocol/bc"
)

type TradePairIterator struct {
	movStore       MovStore
	tradePairs     []*common.TradePair
	tradePairIndex int
}

func NewTradePairIterator(movStore MovStore) *TradePairIterator {
	return &TradePairIterator{movStore: movStore}
}

func (t *TradePairIterator) hasNext() (bool, error) {
	if tradePairSize := len(t.tradePairs); t.tradePairIndex >= tradePairSize {
		var fromAssetID, toAssetID *bc.AssetID
		if len(t.tradePairs) > 0 {
			lastTradePair := t.tradePairs[tradePairSize-1]
			fromAssetID, toAssetID = lastTradePair.FromAssetID, lastTradePair.ToAssetID
		}

		tradePairs, err := t.movStore.ListTradePairsWithStart(fromAssetID, toAssetID)
		if err != nil {
			return false, err
		}

		if len(tradePairs) == 0 {
			return false, nil
		}

		t.tradePairs = tradePairs
		t.tradePairIndex = 0
	}
	return true, nil
}

func (t *TradePairIterator) Next() (*common.TradePair, error) {
	hasNext, err := t.hasNext()
	if err != nil {
		return nil, err
	}

	if !hasNext {
		return nil, nil
	}

	tradePair := t.tradePairs[t.tradePairIndex]
	t.tradePairIndex++
	return tradePair, nil
}

type OrderIterator struct {
	movStore    MovStore
	lastOrder   *common.Order
}

func NewOrderIterator(movStore MovStore, tradePair *common.TradePair) *OrderIterator {
	return &OrderIterator{
		movStore:    movStore,
		lastOrder:   &common.Order{FromAssetID: tradePair.FromAssetID, ToAssetID: tradePair.ToAssetID},
	}
}

func (o *OrderIterator) NextBatch() ([]*common.Order, error) {
	orders, err := o.movStore.ListOrders(o.lastOrder)
	if err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return nil, nil
	}

	o.lastOrder = orders[len(orders)-1]
	return orders, nil
}
