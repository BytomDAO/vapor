package match

import (
	"sort"

	"github.com/vapor/application/mov/common"
	"github.com/vapor/application/mov/database"
	"github.com/vapor/errors"
)

type OrderTable struct {
	movStore    database.MovStore
	orderMap    map[string][]*common.Order
	iteratorMap map[string]*database.OrderIterator

	// tradePair -> []order
	extraAddOrderMap map[string][]*common.Order
	// key of order -> order
	extraDelOrderMap map[string]*common.Order
}

func NewOrderTable(movStore database.MovStore, extraAddOrders, extraDelOrders []*common.Order) *OrderTable {
	return &OrderTable{
		movStore:    movStore,
		orderMap:    make(map[string][]*common.Order),
		iteratorMap: make(map[string]*database.OrderIterator),

		extraAddOrderMap: arrangeExtraAddOrders(extraAddOrders),
		extraDelOrderMap: arrangeExtraDelOrders(extraDelOrders),
	}
}

func (o *OrderTable) PeekOrder(tradePair *common.TradePair) *common.Order {
	if len(o.orderMap[tradePair.Key()]) == 0 {
		o.extendOrders(tradePair)
	}

	var nextOrder *common.Order

	orders := o.orderMap[tradePair.Key()]
	if len(orders) != 0 {
		nextOrder = orders[len(orders) - 1]
	}

	if nextOrder != nil && o.extraDelOrderMap[nextOrder.Key()] != nil {
		o.orderMap[tradePair.Key()] = orders[0 : len(orders)-1]
		delete(o.extraDelOrderMap, nextOrder.Key())
		return o.PeekOrder(tradePair)
	}

	extraOrder := o.peekExtraOrder(tradePair)
	if nextOrder == nil || (extraOrder != nil && extraOrder.Rate < nextOrder.Rate) {
		nextOrder = extraOrder
	}
	return nextOrder
}

func (o *OrderTable) PopOrder(tradePair *common.TradePair) {
	order := o.PeekOrder(tradePair)
	if order == nil {
		return
	}

	orders := o.orderMap[tradePair.Key()]
	if len(orders) != 0 && orders[len(orders) - 1].Key() == order.Key() {
		o.orderMap[tradePair.Key()] = orders[0 : len(orders)-1]
	}

	extraOrders := o.extraAddOrderMap[tradePair.Key()]
	if len(extraOrders) != 0 && orders[len(extraOrders) - 1].Key() == order.Key() {
		o.extraAddOrderMap[tradePair.Key()] = extraOrders[0 : len(extraOrders)-1]
	}
}

func (o *OrderTable) AddOrder(order *common.Order) error {
	tradePair := order.GetTradePair()
	orders := o.orderMap[tradePair.Key()]
	if len(orders) > 0 && order.Rate > orders[len(orders)-1].Rate {
		return errors.New("rate of order must less than the min order in order table")
	}

	o.orderMap[tradePair.Key()] = append(orders, order)
	return nil
}

func (o *OrderTable) extendOrders(tradePair *common.TradePair) {
	iterator, ok := o.iteratorMap[tradePair.Key()]
	if !ok {
		iterator = database.NewOrderIterator(o.movStore, tradePair)
		o.iteratorMap[tradePair.Key()] = iterator
	}

	nextOrders := iterator.NextBatch()
	for i := len(nextOrders) - 1; i >= 0; i-- {
		o.orderMap[tradePair.Key()] = append(o.orderMap[tradePair.Key()], nextOrders[i])
	}
}

func (o *OrderTable) peekExtraOrder(tradePair *common.TradePair) *common.Order {
	extraAddOrders := o.extraAddOrderMap[tradePair.Key()]
	if len(extraAddOrders) > 0 {
		return extraAddOrders[len(extraAddOrders) -1]
	}
	return nil
}

func arrangeExtraAddOrders(orders []*common.Order) map[string][]*common.Order {
	extraAddOrderMap := make(map[string][]*common.Order)
	for _, order := range orders {
		extraAddOrderMap[order.Key()] = append(extraAddOrderMap[order.Key()], order)
	}

	for _, orders := range extraAddOrderMap {
		sort.Sort(common.OrderSlice(orders))
	}
	return extraAddOrderMap
}

func arrangeExtraDelOrders(orders []*common.Order) map[string]*common.Order {
	extraDelOrderMap := make(map[string]*common.Order)
	for _, order := range orders {
		extraDelOrderMap[order.Key()] = order
	}
	return extraDelOrderMap
}
