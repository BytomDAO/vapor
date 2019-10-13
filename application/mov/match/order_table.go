package match

import (
	"container/list"

	"github.com/vapor/application/mov/common"
	"github.com/vapor/application/mov/database"
)

type OrderTable struct {
	buyOrderList      *list.List
	sellOrderList     *list.List
	buyOrderIterator  *database.OrderIterator
	sellOrderIterator *database.OrderIterator
}

func NewOrderTable(movStore *database.MovStore, buyTradePair *common.TradePair, deltaOrderMap map[string]*database.DeltaOrders) *OrderTable {
	sellTradePair := buyTradePair.Reverse()
	buyOrderIterator := database.NewOrderIterator(movStore, buyTradePair, deltaOrderMap[buyTradePair.ID()])
	sellOrderIterator := database.NewOrderIterator(movStore, sellTradePair, deltaOrderMap[sellTradePair.ID()])

	return &OrderTable{
		buyOrderList:      list.New(),
		sellOrderList:     list.New(),
		buyOrderIterator:  buyOrderIterator,
		sellOrderIterator: sellOrderIterator,
	}
}

func (o *OrderTable) PeekOrder() (*common.Order, *common.Order) {
	return o.buyOrderList.Front().Value.(*common.Order), o.sellOrderList.Front().Value.(*common.Order)
}

func (o *OrderTable) PopOrder() {
	o.buyOrderList.Remove(o.buyOrderList.Front())
	o.sellOrderList.Remove(o.sellOrderList.Front())
}

func (o *OrderTable) AddBuyOrder(order *common.Order) {
	o.buyOrderList.PushFront(order)
}

func (o *OrderTable) AddSellOrder(order *common.Order) {
	o.sellOrderList.PushFront(order)
}

func (o *OrderTable) HasNextOrder() bool {
	if o.buyOrderList.Len() == 0 {
		if !o.buyOrderIterator.HasNext() {
			return false
		}

		orders := o.buyOrderIterator.NextBatch()
		for _, order := range orders {
			o.buyOrderList.PushBack(order)
		}
	}

	if o.sellOrderList.Len() == 0 {
		if !o.sellOrderIterator.HasNext() {
			return false
		}

		orders := o.buyOrderIterator.NextBatch()
		for _, order := range orders {
			o.sellOrderList.PushBack(order)
		}
	}
	return true
}
