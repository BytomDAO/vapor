package database

import (
	"fmt"
	"sort"

	"github.com/vapor/application/mov/common"
	"github.com/vapor/protocol/bc"
)

type TradePairIterator struct {
	movStore       *MovStore
	tradePairs     []*common.TradePair
	tradePairIndex int
}

func NewTradePairIterator(movStore *MovStore) *TradePairIterator {
	return &TradePairIterator{movStore: movStore}
}

func (t *TradePairIterator) HasNext() bool {
	// TODO tradePair重复的过滤
	tradePairSize := len(t.tradePairs)
	if t.tradePairIndex >= tradePairSize {
		var fromAssetID, toAssetID *bc.AssetID
		if len(t.tradePairs) > 0 {
			lastTradePair := t.tradePairs[tradePairSize-1]
			fromAssetID, toAssetID = lastTradePair.FromAssetID, lastTradePair.ToAssetID
		}

		tradePairs, err := t.movStore.ListTradePairsWithStart(fromAssetID, toAssetID)
		if err != nil || len(tradePairs) == 0 {
			// TODO log or return error
			return false
		}

		t.tradePairs = tradePairs
		t.tradePairIndex = 0
	}
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
	movStore    *MovStore
	deltaOrders *DeltaOrders
	lastOrder   *common.Order
	orders      []*common.Order
}

func NewOrderIterator(movStore *MovStore, tradePair *common.TradePair, deltaOrders *DeltaOrders) *OrderIterator {
	return &OrderIterator{
		movStore:    movStore,
		deltaOrders: deltaOrders,
		lastOrder:   &common.Order{FromAssetID: tradePair.FromAssetID, ToAssetID: tradePair.ToAssetID},
	}
}

func (o *OrderIterator) HasNext() bool {
	if o.orders == nil {
		orders, err := o.movStore.ListOrders(o.lastOrder)
		if err != nil || len(orders) == 0 {
			// TODO log or return err?
			return false
		}

		if o.deltaOrders != nil {
			orders = o.deltaOrders.mergeOrders(orders)
		}

		o.orders = orders
		o.lastOrder = o.orders[len(o.orders) - 1]
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

type DeltaOrders struct {
	AddOrders    []*common.Order
	DeleteOrders []*common.Order
}

func (d *DeltaOrders) mergeOrders(orders []*common.Order) []*common.Order {
	var tempOrders, newAddOrders, newDeleteOrders []*common.Order
	tempOrders = append(tempOrders, orders...)

	for _, addOrder := range d.AddOrders {
		if addOrder.Rate <= orders[len(orders) - 1].Rate {
			tempOrders = append(tempOrders, addOrder)
		} else {
			newAddOrders = append(newAddOrders, addOrder)
		}
	}

	deleteOrderMap := make(map[string]*common.Order)
	for _, deleteOrder := range d.DeleteOrders {
		key := fmt.Sprintf("%s:%d", deleteOrder.Utxo.SourceID, deleteOrder.Utxo.SourcePos)
		deleteOrderMap[key] = deleteOrder
	}

	var result []*common.Order
	for _, order := range tempOrders {
		key := fmt.Sprintf("%s:%d", order.Utxo.SourceID, order.Utxo.SourcePos)
		if deleteOrderMap[key] == nil {
			result = append(result, order)
		} else {
			delete(deleteOrderMap, key)
		}
	}

	for _, deleteOrder := range deleteOrderMap {
		newDeleteOrders = append(newDeleteOrders, deleteOrder)
	}

	d.AddOrders = newAddOrders
	d.DeleteOrders = newDeleteOrders
	sort.Sort(common.OrderSlice(result))
	return result
}
