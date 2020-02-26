package mov

import (
	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/application/mov/database"
)

func queryAllOrders(store *database.LevelDBMovStore) []*common.Order {
	var orders []*common.Order
	tradePairIterator := database.NewTradePairIterator(store)
	for tradePairIterator.HasNext() {
		orderIterator := database.NewOrderIterator(store, tradePairIterator.Next())
		for orderIterator.HasNext() {
			orders = append(orders, orderIterator.NextBatch()...)
		}
	}
	return orders
}

// QueryAllOrders query all orders from db
func QueryAllOrders(store *database.LevelDBMovStore) []*common.Order {
	return queryAllOrders(store)
}
