package match

import (
	"github.com/vapor/application/mov/common"
	"github.com/vapor/application/mov/database"
	"github.com/vapor/errors"
)

type OrderTable struct {
	movStore    database.MovStore
	orderMap    map[string][]*common.Order
	iteratorMap map[string]*database.OrderIterator
}

func NewOrderTable(movStore database.MovStore) *OrderTable {
	return &OrderTable{
		movStore:    movStore,
		orderMap:    make(map[string][]*common.Order),
		iteratorMap: make(map[string]*database.OrderIterator),
	}
}

func (o *OrderTable) PeekOrder(tradePair *common.TradePair) (*common.Order, error) {
	if len(o.orderMap[tradePair.Key()]) == 0 {
		iterator := o.iteratorMap[tradePair.Key()]
		if iterator == nil {
			iterator = database.NewOrderIterator(o.movStore, tradePair)
			o.iteratorMap[tradePair.Key()] = iterator
		}

		orders, err := iterator.NextBatch()
		if err != nil {
			return nil, err
		}

		if len(orders) == 0 {
			return nil, nil
		}

		for i := len(orders) - 1; i >= 0; i-- {
			o.orderMap[tradePair.Key()] = append(o.orderMap[tradePair.Key()], orders[i])
		}
	}
	orders := o.orderMap[tradePair.Key()]
	return orders[len(orders)-1], nil
}

func (o *OrderTable) PopOrder(tradePair *common.TradePair) {
	orders := o.orderMap[tradePair.Key()]
	if len(orders) > 0 {
		o.orderMap[tradePair.Key()] = orders[0 : len(orders)-1]
	}
}

func (o *OrderTable) AddOrder(tradePair *common.TradePair, order *common.Order) error {
	orders := o.orderMap[tradePair.Key()]
	if len(orders) > 0 && order.Rate > orders[len(orders)-1].Rate {
		return errors.New("rate of order must less than the min order in order table")
	}

	o.orderMap[tradePair.Key()] = append(orders, order)
	return nil
}
