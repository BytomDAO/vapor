package match

import (
	"github.com/vapor/application/mov/common"
	"github.com/vapor/application/mov/database"
	"github.com/vapor/errors"
)

var errOrderRate = errors.New("rate of order must less than the min order in order table")

type OrderTable struct {
	buyOrders         []*common.Order
	sellOrders        []*common.Order
	buyOrderIterator  *database.OrderIterator
	sellOrderIterator *database.OrderIterator
}

func NewOrderTable(movStore *database.MovStore, buyTradePair *common.TradePair, deltaOrderMap map[string]*database.DeltaOrders) *OrderTable {
	sellTradePair := buyTradePair.Reverse()
	buyOrderIterator := database.NewOrderIterator(movStore, buyTradePair, deltaOrderMap[buyTradePair.String()])
	sellOrderIterator := database.NewOrderIterator(movStore, sellTradePair, deltaOrderMap[sellTradePair.String()])

	return &OrderTable{
		buyOrderIterator:  buyOrderIterator,
		sellOrderIterator: sellOrderIterator,
	}
}

func (o *OrderTable) PeekOrder() (*common.Order, *common.Order) {
	if !o.HasNextOrder() {
		return nil, nil
	}
	return o.buyOrders[len(o.buyOrders)-1], o.sellOrders[len(o.sellOrders)-1]
}

func (o *OrderTable) PopOrder() {
	o.buyOrders = o.buyOrders[0 : len(o.buyOrders)-1]
	o.sellOrders = o.sellOrders[0: len(o.sellOrders)-1]
}

func (o *OrderTable) AddBuyOrder(order *common.Order) error {
	if len(o.buyOrders) > 0 && order.Rate > o.buyOrders[len(o.buyOrders) - 1].Rate {
		return errOrderRate
	}
	o.buyOrders = append(o.buyOrders, order)
	return nil
}

func (o *OrderTable) AddSellOrder(order *common.Order) error {
	if len(o.sellOrders) > 0 && order.Rate > o.sellOrders[len(o.sellOrders) - 1].Rate {
		return errOrderRate
	}
	o.sellOrders = append(o.sellOrders, order)
	return nil
}

func (o *OrderTable) HasNextOrder() bool {
	if len(o.buyOrders) == 0 {
		if !o.buyOrderIterator.HasNext() {
			return false
		}

		orders := o.buyOrderIterator.NextBatch()
		for i := len(orders) - 1; i >= 0; i-- {
			o.buyOrders = append(o.buyOrders, orders[i])
		}
	}

	if len(o.sellOrders) == 0 {
		if !o.sellOrderIterator.HasNext() {
			return false
		}

		orders := o.sellOrderIterator.NextBatch()
		for i := len(orders) - 1; i >= 0; i-- {
			o.sellOrders = append(o.sellOrders, orders[i])
		}
	}
	return true
}
