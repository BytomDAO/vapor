package database

import (
	log "github.com/sirupsen/logrus"
	"github.com/vapor/application/mov/common"
	"github.com/vapor/protocol/bc"
)

type TradePairIterator struct {
	movStore       *MovStore
	tradePairs     []*common.TradePair
	tradePairMap   map[string]bool
	tradePairIndex int
}

func NewTradePairIterator(movStore *MovStore) *TradePairIterator {
	return &TradePairIterator{movStore: movStore, tradePairMap: make(map[string]bool)}
}

func (t *TradePairIterator) HasNext() bool {
	if tradePairSize := len(t.tradePairs); t.tradePairIndex >= tradePairSize {
		var fromAssetID, toAssetID *bc.AssetID
		if len(t.tradePairs) > 0 {
			lastTradePair := t.tradePairs[tradePairSize-1]
			fromAssetID, toAssetID = lastTradePair.FromAssetID, lastTradePair.ToAssetID
		}

		tradePairs, err := t.movStore.ListTradePairsWithStart(fromAssetID, toAssetID)
		if err != nil {
			log.WithField("err", err).Error("fail to list trade pair")
			return false
		}

		if len(tradePairs) == 0 {
			return false
		}

		t.tradePairs = tradePairs
		t.tradePairIndex = 0
	}

	if t.tradePairMap[t.tradePairs[t.tradePairIndex].String()] {
		t.tradePairIndex++
		return t.HasNext()
	}
	return true
}

func (t *TradePairIterator) Next() *common.TradePair {
	if !t.HasNext() {
		return nil
	}

	tradePair := t.tradePairs[t.tradePairIndex]
	t.tradePairMap[tradePair.String()] = true
	t.tradePairMap[tradePair.Reverse().String()] = true
	t.tradePairIndex++
	return tradePair
}

type OrderIterator struct {
	movStore    *MovStore
	lastOrder   *common.Order
	orders      []*common.Order
}

func NewOrderIterator(movStore *MovStore, tradePair *common.TradePair) *OrderIterator {
	return &OrderIterator{
		movStore:    movStore,
		lastOrder:   &common.Order{FromAssetID: tradePair.FromAssetID, ToAssetID: tradePair.ToAssetID},
	}
}

func (o *OrderIterator) HasNext() bool {
	if len(o.orders) == 0 {
		orders, err := o.movStore.ListOrders(o.lastOrder)
		if err != nil {
			log.WithField("err", err).Error("fail to list orders")
			return false
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
