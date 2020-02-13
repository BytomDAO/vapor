package match

import (
	"sort"
	"sync"

	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/application/mov/database"
)

// OrderBook is used to handle the mov orders in memory like stack
type OrderBook struct {
	movStore database.MovStore
	// key of tradePair -> []order
	dbOrders *sync.Map
	// key of tradePair -> iterator
	orderIterators *sync.Map

	// key of tradePair -> []order
	arrivalAddOrders *sync.Map
	// key of order -> order
	arrivalDelOrders *sync.Map
}

func arrangeArrivalAddOrders(orders []*common.Order) *sync.Map {
	orderMap := make(map[string][]*common.Order)
	for _, order := range orders {
		orderMap[order.TradePair().Key()] = append(orderMap[order.TradePair().Key()], order)
	}

	arrivalOrderMap := &sync.Map{}
	for key, orders := range orderMap {
		sort.Sort(sort.Reverse(common.OrderSlice(orders)))
		arrivalOrderMap.Store(key, orders)
	}
	return arrivalOrderMap
}

func arrangeArrivalDelOrders(orders []*common.Order) *sync.Map {
	arrivalDelOrderMap := &sync.Map{}
	for _, order := range orders {
		arrivalDelOrderMap.Store(order.Key(), order)
	}
	return arrivalDelOrderMap
}

// NewOrderBook create a new OrderBook object
func NewOrderBook(movStore database.MovStore, arrivalAddOrders, arrivalDelOrders []*common.Order) *OrderBook {
	return &OrderBook{
		movStore:       movStore,
		dbOrders:       &sync.Map{},
		orderIterators: &sync.Map{},

		arrivalAddOrders: arrangeArrivalAddOrders(arrivalAddOrders),
		arrivalDelOrders: arrangeArrivalDelOrders(arrivalDelOrders),
	}
}

// AddOrder add the in memory temp order to order table, because temp order is what left for the
// partial trade order, so the price should be lowest.
func (o *OrderBook) AddOrder(order *common.Order) {
	tradePairKey := order.TradePair().Key()
	orders := o.getArrivalAddOrders(tradePairKey)
	// use binary search to find the insert position
	i := sort.Search(len(orders), func(i int) bool { return order.Cmp(orders[i]) > 0 })
	orders = append(orders, order)
	copy(orders[i+1:], orders[i:])
	orders[i] = order

	o.arrivalAddOrders.Store(tradePairKey, orders)
}

// DelOrder mark the order has been deleted in order book
func (o *OrderBook) DelOrder(order *common.Order) {
	o.arrivalDelOrders.Store(order.Key(), order)
}

// PeekOrder return the next lowest order of given trade pair
func (o *OrderBook) PeekOrder(tradePair *common.TradePair) *common.Order {
	if len(o.getDBOrders(tradePair.Key())) == 0 {
		o.extendDBOrders(tradePair)
	}

	var nextOrder *common.Order
	orders := o.getDBOrders(tradePair.Key())
	if len(orders) != 0 {
		nextOrder = orders[len(orders)-1]
	}

	if nextOrder != nil && o.getArrivalDelOrders(nextOrder.Key()) != nil {
		o.dbOrders.Store(tradePair.Key(), orders[0:len(orders)-1])
		return o.PeekOrder(tradePair)
	}

	arrivalOrder := o.peekArrivalOrder(tradePair)
	if nextOrder == nil || (arrivalOrder != nil && arrivalOrder.Cmp(nextOrder) < 0) {
		nextOrder = arrivalOrder
	}
	return nextOrder
}

// PeekOrders return the next lowest orders by given array of trade pairs
func (o *OrderBook) PeekOrders(tradePairs []*common.TradePair) []*common.Order {
	var orders []*common.Order
	for _, tradePair := range tradePairs {
		order := o.PeekOrder(tradePair)
		if order == nil {
			return nil
		}

		orders = append(orders, order)
	}
	return orders
}

// PopOrder delete the next lowest order of given trade pair
func (o *OrderBook) PopOrder(tradePair *common.TradePair) {
	order := o.PeekOrder(tradePair)
	if order == nil {
		return
	}

	orders := o.getDBOrders(tradePair.Key())
	if len(orders) != 0 && orders[len(orders)-1].Key() == order.Key() {
		o.dbOrders.Store(tradePair.Key(), orders[0:len(orders)-1])
		return
	}

	arrivalOrders := o.getArrivalAddOrders(tradePair.Key())
	if len(arrivalOrders) != 0 && arrivalOrders[len(arrivalOrders)-1].Key() == order.Key() {
		o.arrivalAddOrders.Store(tradePair.Key(), arrivalOrders[0:len(arrivalOrders)-1])
	}
}

// PopOrders delete the next lowest orders by given trade pairs
func (o *OrderBook) PopOrders(tradePairs []*common.TradePair) []*common.Order {
	var orders []*common.Order
	for _, tradePair := range tradePairs {
		o.PopOrder(tradePair)
	}
	return orders
}

func (o *OrderBook) getDBOrders(tradePairKey string) []*common.Order {
	if orders, ok := o.dbOrders.Load(tradePairKey); ok {
		return orders.([]*common.Order)
	}
	return []*common.Order{}
}

func (o *OrderBook) getArrivalAddOrders(tradePairKey string) []*common.Order {
	if orders, ok := o.arrivalAddOrders.Load(tradePairKey); ok {
		return orders.([]*common.Order)
	}
	return []*common.Order{}
}

func (o *OrderBook) getArrivalDelOrders(orderKey string) *common.Order {
	if order, ok := o.arrivalDelOrders.Load(orderKey); ok {
		return order.(*common.Order)
	}
	return nil
}

func (o *OrderBook) extendDBOrders(tradePair *common.TradePair) {
	iterator, ok := o.orderIterators.Load(tradePair.Key())
	if !ok {
		iterator = database.NewOrderIterator(o.movStore, tradePair)
		o.orderIterators.Store(tradePair.Key(), iterator)
	}

	nextOrders := iterator.(*database.OrderIterator).NextBatch()
	orders := o.getDBOrders(tradePair.Key())
	for i := len(nextOrders) - 1; i >= 0; i-- {
		orders = append(orders, nextOrders[i])
	}
	o.dbOrders.Store(tradePair.Key(), orders)
}

func (o *OrderBook) peekArrivalOrder(tradePair *common.TradePair) *common.Order {
	orders := o.getArrivalAddOrders(tradePair.Key())
	delPos := len(orders)
	for i := delPos - 1; i >= 0; i-- {
		if o.getArrivalDelOrders(orders[i].Key()) != nil {
			delPos = i
		} else {
			break
		}
	}

	if delPos < len(orders) {
		orders = orders[:delPos]
		o.arrivalAddOrders.Store(tradePair.Key(), orders)
	}

	if size := len(orders); size > 0 {
		return orders[size-1]
	}
	return nil
}
