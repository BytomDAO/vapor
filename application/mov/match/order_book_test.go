package match

import (
	"testing"

	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/application/mov/database"
	"github.com/bytom/vapor/application/mov/mock"
)

var (
	btc2eth = &common.TradePair{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH}
)

func TestOrderBook(t *testing.T) {
	cases := []struct {
		desc                 string
		initMovStore         database.MovStore
		initArrivalAddOrders []*common.Order
		initArrivalDelOrders []*common.Order
		addOrders            []*common.Order
		popOrders            []*common.TradePair
		wantPeekedOrders     map[common.TradePair]*common.Order
	}{
		{
			desc: "no arrival orders, no add order, no pop order",
			initMovStore: mock.NewMovStore(
				[]*common.TradePair{btc2eth},
				[]*common.Order{
					mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Btc2EthOrders[2],
				},
			),
			wantPeekedOrders: map[common.TradePair]*common.Order{
				*btc2eth: mock.Btc2EthOrders[0],
			},
		},
		{
			desc: "no arrival orders, add lower price order, no pop order",
			initMovStore: mock.NewMovStore(
				[]*common.TradePair{btc2eth},
				[]*common.Order{
					mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Btc2EthOrders[2],
				}),
			addOrders: []*common.Order{mock.Btc2EthOrders[3]},
			wantPeekedOrders: map[common.TradePair]*common.Order{
				*btc2eth: mock.Btc2EthOrders[3],
			},
		},
		{
			desc: "no arrival orders, no add order, pop one order",
			initMovStore: mock.NewMovStore(
				[]*common.TradePair{btc2eth},
				[]*common.Order{
					mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Btc2EthOrders[2],
				}),
			popOrders: []*common.TradePair{btc2eth},
			wantPeekedOrders: map[common.TradePair]*common.Order{
				*btc2eth: mock.Btc2EthOrders[2],
			},
		},
		{
			desc: "has arrival add orders, no add order, no pop order, the arrival add order is lower price",
			initMovStore: mock.NewMovStore(
				[]*common.TradePair{btc2eth},
				[]*common.Order{
					mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Btc2EthOrders[2],
				}),
			initArrivalAddOrders: []*common.Order{mock.Btc2EthOrders[3]},
			wantPeekedOrders: map[common.TradePair]*common.Order{
				*btc2eth: mock.Btc2EthOrders[3],
			},
		},
		{
			desc: "has arrival add orders, no add order, no pop order, the db add order is lower price",
			initMovStore: mock.NewMovStore(
				[]*common.TradePair{btc2eth},
				[]*common.Order{
					mock.Btc2EthOrders[0], mock.Btc2EthOrders[1],
				}),
			initArrivalAddOrders: []*common.Order{mock.Btc2EthOrders[2]},
			wantPeekedOrders: map[common.TradePair]*common.Order{
				*btc2eth: mock.Btc2EthOrders[0],
			},
		},
		{
			desc: "has arrival add orders, no add order, pop one order, after pop the arrival order is lower price",
			initMovStore: mock.NewMovStore(
				[]*common.TradePair{btc2eth},
				[]*common.Order{
					mock.Btc2EthOrders[1], mock.Btc2EthOrders[2], mock.Btc2EthOrders[3],
				}),
			initArrivalAddOrders: []*common.Order{mock.Btc2EthOrders[0]},
			popOrders:            []*common.TradePair{btc2eth},
			wantPeekedOrders: map[common.TradePair]*common.Order{
				*btc2eth: mock.Btc2EthOrders[0],
			},
		},
		{
			desc: "has arrival delete orders, no add order, no pop order",
			initMovStore: mock.NewMovStore(
				[]*common.TradePair{btc2eth},
				[]*common.Order{
					mock.Btc2EthOrders[1], mock.Btc2EthOrders[2], mock.Btc2EthOrders[3],
				}),
			initArrivalDelOrders: []*common.Order{mock.Btc2EthOrders[3]},
			wantPeekedOrders: map[common.TradePair]*common.Order{
				*btc2eth: mock.Btc2EthOrders[2],
			},
		},
		{
			desc: "has arrival delete orders and arrival add orders, no add order, no pop order, the arrival order is lower price",
			initMovStore: mock.NewMovStore(
				[]*common.TradePair{btc2eth},
				[]*common.Order{
					mock.Btc2EthOrders[3], mock.Btc2EthOrders[1], mock.Btc2EthOrders[2],
				}),
			initArrivalAddOrders: []*common.Order{mock.Btc2EthOrders[0]},
			initArrivalDelOrders: []*common.Order{mock.Btc2EthOrders[3]},
			wantPeekedOrders: map[common.TradePair]*common.Order{
				*btc2eth: mock.Btc2EthOrders[0],
			},
		},
		{
			desc: "has arrival delete orders and arrival add orders, no add order, pop one order",
			initMovStore: mock.NewMovStore(
				[]*common.TradePair{btc2eth},
				[]*common.Order{
					mock.Btc2EthOrders[3], mock.Btc2EthOrders[1],
				}),
			initArrivalAddOrders: []*common.Order{mock.Btc2EthOrders[0], mock.Btc2EthOrders[2]},
			initArrivalDelOrders: []*common.Order{mock.Btc2EthOrders[3]},
			popOrders:            []*common.TradePair{btc2eth},
			wantPeekedOrders: map[common.TradePair]*common.Order{
				*btc2eth: mock.Btc2EthOrders[2],
			},
		},
		{
			desc: "has arrival add orders, but db order is empty",
			initMovStore: mock.NewMovStore(
				[]*common.TradePair{btc2eth},
				[]*common.Order{}),
			initArrivalAddOrders: []*common.Order{mock.Btc2EthOrders[0], mock.Btc2EthOrders[2]},
			wantPeekedOrders: map[common.TradePair]*common.Order{
				*btc2eth: mock.Btc2EthOrders[0],
			},
		},
		{
			desc: "no add orders, and db order is empty",
			initMovStore: mock.NewMovStore(
				[]*common.TradePair{btc2eth},
				[]*common.Order{}),
			initArrivalAddOrders: []*common.Order{},
			wantPeekedOrders: map[common.TradePair]*common.Order{
				*btc2eth: nil,
			},
		},
		{
			desc: "has arrival delete orders, no add order, no pop order, need recursive to peek one order",
			initMovStore: mock.NewMovStore(
				[]*common.TradePair{btc2eth},
				[]*common.Order{
					mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Btc2EthOrders[2], mock.Btc2EthOrders[3],
				}),
			initArrivalAddOrders: []*common.Order{},
			initArrivalDelOrders: []*common.Order{mock.Btc2EthOrders[3], mock.Btc2EthOrders[0], mock.Btc2EthOrders[2]},
			wantPeekedOrders: map[common.TradePair]*common.Order{
				*btc2eth: mock.Btc2EthOrders[1],
			},
		},
	}

	for i, c := range cases {
		orderBook := NewOrderBook(c.initMovStore, c.initArrivalAddOrders, c.initArrivalDelOrders)
		for _, order := range c.addOrders {
			if err := orderBook.AddOrder(order); err != nil {
				t.Fatal(err)
			}
		}

		for _, tradePair := range c.popOrders {
			orderBook.PopOrder(tradePair)
		}

		for tradePair, wantOrder := range c.wantPeekedOrders {
			gotOrder := orderBook.PeekOrder(&tradePair)
			if wantOrder == gotOrder && wantOrder == nil {
				continue
			}

			if gotOrder.Key() != wantOrder.Key() {
				t.Errorf("#%d(%s):the key of got order(%v) is not equals key of want order(%v)", i, c.desc, gotOrder, wantOrder)
			}
		}
	}
}
