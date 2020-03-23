package database

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/protocol/bc"
)

// TradePairIterator wrap read trade pair from DB action
type TradePairIterator struct {
	movStore       MovStore
	tradePairs     []*common.TradePair
	tradePairIndex int
}

// NewTradePairIterator create the new TradePairIterator object
func NewTradePairIterator(movStore MovStore) *TradePairIterator {
	return &TradePairIterator{movStore: movStore}
}

// HasNext check if there are more trade pairs in memory or DB
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

// Next return the next available trade pair in memory or DB
func (t *TradePairIterator) Next() *common.TradePair {
	if !t.HasNext() {
		return nil
	}

	tradePair := t.tradePairs[t.tradePairIndex]
	t.tradePairIndex++
	return tradePair
}

// OrderIterator wrap read order from DB action
type OrderIterator struct {
	movStore  MovStore
	lastOrder *common.Order
	orders    []*common.Order
}

// NewOrderIterator create the new OrderIterator object
func NewOrderIterator(movStore MovStore, tradePair *common.TradePair) *OrderIterator {
	return &OrderIterator{
		movStore:  movStore,
		lastOrder: &common.Order{FromAssetID: tradePair.FromAssetID, ToAssetID: tradePair.ToAssetID},
	}
}

// HasNext check if there are more orders in memory or DB
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

// NextBatch return the next batch of orders in memory or DB
func (o *OrderIterator) NextBatch() []*common.Order {
	if !o.HasNext() {
		return nil
	}

	orders := o.orders
	o.orders = nil
	return orders
}
