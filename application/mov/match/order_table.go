package match

import (
	"sort"

	"github.com/vapor/application/mov/common"
	"github.com/vapor/application/mov/database"
	"github.com/vapor/errors"
)

type OrderTable struct {
	movStore       database.MovStore
	// key of tradePair -> []order
	dbOrders map[string][]*common.Order
	// key of tradePair -> iterator
	orderIterators map[string]*database.OrderIterator

	// key of tradePair -> []order
	arrivalAddOrders map[string][]*common.Order
	// key of order -> order
	arrivalDelOrders map[string]*common.Order
}

func NewOrderTable(movStore database.MovStore, arrivalAddOrders, arrivalDelOrders []*common.Order) *OrderTable {
	return &OrderTable{
		movStore:       movStore,
		dbOrders:       make(map[string][]*common.Order),
		orderIterators: make(map[string]*database.OrderIterator),

		arrivalAddOrders: arrangeArrivalAddOrders(arrivalAddOrders),
		arrivalDelOrders: arrangeArrivalDelOrders(arrivalDelOrders),
	}
}

func (o *OrderTable) PeekOrder(tradePair *common.TradePair) *common.Order {
	if len(o.dbOrders[tradePair.Key()]) == 0 {
		o.extendDBOrders(tradePair)
	}

	var nextOrder *common.Order

	orders := o.dbOrders[tradePair.Key()]
	if len(orders) != 0 {
		nextOrder = orders[len(orders) - 1]
	}

	if nextOrder != nil && o.arrivalDelOrders[nextOrder.Key()] != nil {
		o.dbOrders[tradePair.Key()] = orders[0 : len(orders)-1]
		delete(o.arrivalDelOrders, nextOrder.Key())
		return o.PeekOrder(tradePair)
	}

	arrivalOrder := o.peekArrivalOrder(tradePair)
	if nextOrder == nil || (arrivalOrder != nil && arrivalOrder.Rate < nextOrder.Rate) {
		nextOrder = arrivalOrder
	}
	return nextOrder
}

func (o *OrderTable) PopOrder(tradePair *common.TradePair) {
	order := o.PeekOrder(tradePair)
	if order == nil {
		return
	}

	orders := o.dbOrders[tradePair.Key()]
	if len(orders) != 0 && orders[len(orders) - 1].Key() == order.Key() {
		o.dbOrders[tradePair.Key()] = orders[0 : len(orders)-1]
	}

	arrivalOrders := o.arrivalAddOrders[tradePair.Key()]
	if len(arrivalOrders) != 0 && arrivalOrders[len(arrivalOrders) - 1].Key() == order.Key() {
		o.arrivalAddOrders[tradePair.Key()] = arrivalOrders[0 : len(arrivalOrders)-1]
	}
}

func (o *OrderTable) AddOrder(order *common.Order) error {
	tradePair := order.TradePair()
	orders := o.dbOrders[tradePair.Key()]
	if len(orders) > 0 && order.Rate > orders[len(orders)-1].Rate {
		return errors.New("rate of order must less than the min order in order table")
	}

	o.dbOrders[tradePair.Key()] = append(orders, order)
	return nil
}

func (o *OrderTable) extendDBOrders(tradePair *common.TradePair) {
	iterator, ok := o.orderIterators[tradePair.Key()]
	if !ok {
		iterator = database.NewOrderIterator(o.movStore, tradePair)
		o.orderIterators[tradePair.Key()] = iterator
	}

	nextOrders := iterator.NextBatch()
	for i := len(nextOrders) - 1; i >= 0; i-- {
		o.dbOrders[tradePair.Key()] = append(o.dbOrders[tradePair.Key()], nextOrders[i])
	}
}

func (o *OrderTable) peekArrivalOrder(tradePair *common.TradePair) *common.Order {
	arrivalAddOrders := o.arrivalAddOrders[tradePair.Key()]
	if len(arrivalAddOrders) > 0 {
		return arrivalAddOrders[len(arrivalAddOrders) -1]
	}
	return nil
}

func arrangeArrivalAddOrders(orders []*common.Order) map[string][]*common.Order {
	arrivalAddOrderMap := make(map[string][]*common.Order)
	for _, order := range orders {
		arrivalAddOrderMap[order.TradePair().Key()] = append(arrivalAddOrderMap[order.TradePair().Key()], order)
	}

	for _, orders := range arrivalAddOrderMap {
		sort.Sort(sort.Reverse(common.OrderSlice(orders)))
	}
	return arrivalAddOrderMap
}

func arrangeArrivalDelOrders(orders []*common.Order) map[string]*common.Order {
	arrivalDelOrderMap := make(map[string]*common.Order)
	for _, order := range orders {
		arrivalDelOrderMap[order.Key()] = order
	}
	return arrivalDelOrderMap
}
