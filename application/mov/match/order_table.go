package match

import (
	"fmt"

	"github.com/vapor/application/mov/common"
	"github.com/vapor/application/mov/database"
	vprCommon "github.com/vapor/common"
)

type OrderTable struct {
	buyOrderStack     *vprCommon.Stack
	sellOrderStack    *vprCommon.Stack
	buyOrderIterator  *database.OrderIterator
	sellOrderIterator *database.OrderIterator
}

func NewOrderTable(movStore *database.MovStore, tradePair *common.TradePair, deltaOrderMap map[string]*database.DeltaOrders) *OrderTable {
	buyTradePairKey := fmt.Sprintf("%s:%s", tradePair.FromAssetID, tradePair.ToAssetID)
	sellTradePairKey := fmt.Sprintf("%s:%s", tradePair.ToAssetID, tradePair.FromAssetID)
	sellTradePair := &common.TradePair{FromAssetID:tradePair.ToAssetID, ToAssetID:tradePair.FromAssetID}

	return &OrderTable{
		buyOrderStack:  vprCommon.NewStack(),
		sellOrderStack: vprCommon.NewStack(),
		buyOrderIterator: database.NewOrderIterator(movStore, tradePair, deltaOrderMap[buyTradePairKey]),
		sellOrderIterator: database.NewOrderIterator(movStore, sellTradePair, deltaOrderMap[sellTradePairKey]),
	}
}

func (o *OrderTable) PeekOrder() (*common.Order, *common.Order) {
	return o.buyOrderStack.Peek().(*common.Order), o.sellOrderStack.Peek().(*common.Order)
}

func (o *OrderTable) PopOrder() {
	o.buyOrderStack.Pop()
	o.sellOrderStack.Pop()
}

func (o *OrderTable) AddBuyOrder(order *common.Order) {
	o.buyOrderStack.Push(order)
}

func (o *OrderTable) AddSellOrder(order *common.Order) {
	o.sellOrderStack.Push(order)
}

func (o *OrderTable) HasNextOrder() bool {
	if o.buyOrderStack.Len() == 0 {
		if !o.buyOrderIterator.HasNext() {
			return false
		}

		orders := o.buyOrderIterator.NextBatch()
		for i := len(orders) - 1; i >= 0; i-- {
			o.buyOrderStack.Push(orders[i])
		}
	}

	if o.sellOrderStack.Len() == 0 {
		if !o.sellOrderIterator.HasNext() {
			return false
		}

		orders := o.sellOrderIterator.NextBatch()
		for i := len(orders) - 1; i >= 0; i-- {
			o.sellOrderStack.Push(orders[i])
		}
	}
	return true
}
