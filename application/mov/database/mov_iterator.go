package database

import (
	"container/list"
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

type DeltaOrders struct {
	addOrders    *list.List
	deleteOrders map[string]*common.Order
}

func NewDeltaOrders() *DeltaOrders {
	return &DeltaOrders{
		addOrders:    list.New(),
		deleteOrders: make(map[string]*common.Order),
	}
}

func (d *DeltaOrders) AppendAddOrder(orders... *common.Order) {
	for _, order := range orders {
		d.addOrders.PushBack(order)
	}
}

func (d *DeltaOrders) AppendDeleteOrder(orders... *common.Order) {
	for _, order := range orders {
		d.deleteOrders[order.ID()] = order
	}
}

func (d *DeltaOrders) mergeOrders(orders []*common.Order) []*common.Order {
	orderList := list.New()
	for _, order := range orders {
		orderList.PushBack(order)
	}

	for element := d.addOrders.Front(); element != nil; {
		next := element.Next()
		addOrder := element.Value.(*common.Order)
		if addOrder.Rate <= orders[len(orders)-1].Rate {
			orderList.PushBack(addOrder)
			d.addOrders.Remove(element)
		}
		element = next
	}

	for element := orderList.Front(); element != nil; {
		next := element.Next()
		order := element.Value.(*common.Order)
		if d.deleteOrders[order.ID()] != nil {
			orderList.Remove(element)
			delete(d.deleteOrders, order.ID())
		}
		element = next
	}

	var result []*common.Order
	for element := orderList.Front(); element != nil; element = element.Next() {
		result = append(result, element.Value.(*common.Order))
	}
	sort.Sort(common.OrderSlice(result))
	return result
}
