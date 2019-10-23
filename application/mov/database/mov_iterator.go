package database

import (
	log "github.com/sirupsen/logrus"

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

func (t *TradePairIterator) HasNext() bool {
	tradePairSize := len(t.tradePairs)
	if t.tradePairIndex < tradePairSize {
		return true
	}
	var fromAssetID, toAssetID *bc.AssetID
	if len(t.tradePairs) > 0 {
		lastTradePair := t.tradePairs[tradePairSize-1]
		fromAssetID, toAssetID = lastTradePair.FromAssetID, lastTradePair.ToAssetID
	}

	tradePairs, err := t.movStore.ListTradePairsWithStart(fromAssetID, toAssetID)
	if err != nil {
		// If the error is returned, it's an error of program itself,
		// and cannot be recovered, so panic directly.
		log.WithField("err", err).Fatal("fail to list trade pairs")
	}

	if len(tradePairs) == 0 {
		return false
	}

	t.tradePairs = tradePairs
	t.tradePairIndex = 0
	return true
}

func (t *TradePairIterator) Next() *common.TradePair {
	if !t.HasNext() {
		return nil
	}

	tradePair := t.tradePairs[t.tradePairIndex]
	t.tradePairIndex++
	return tradePair
}

type OrderIterator struct {
	movStore  MovStore
	lastOrder *common.Order
	orders    []*common.Order
}

func NewOrderIterator(movStore MovStore, tradePair *common.TradePair) *OrderIterator {
	return &OrderIterator{
		movStore:  movStore,
		lastOrder: &common.Order{FromAssetID: tradePair.FromAssetID, ToAssetID: tradePair.ToAssetID},
	}
}

func (o *OrderIterator) HasNext() bool {
	if len(o.orders) == 0 {
		orders, err := o.movStore.ListOrders(o.lastOrder)
		if err != nil {
			log.WithField("err", err).Fatal("fail to list orders")
		}

		if len(orders) == 0 {
			return false
		}

		o.orders = orders
		o.lastOrder = o.orders[len(o.orders)-1]
	}
	return true
}

func (o *OrderIterator) NextBatch() []*common.Order {
	if !o.HasNext() {
		return nil
	}

	orders := o.orders
	o.orders = nil
	return orders
}
