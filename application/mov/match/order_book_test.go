package match

import (
	"testing"

	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/application/mov/database"
	"github.com/bytom/vapor/application/mov/mock"
	"github.com/bytom/vapor/testutil"
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
			orderBook.AddOrder(order)
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
				t.Fatalf("#%d(%s):the key of got order(%v) is not equals key of want order(%v)", i, c.desc, gotOrder, wantOrder)
			}
		}
	}
}

func TestPeekArrivalOrder(t *testing.T) {
	cases := []struct {
		desc                 string
		initArrivalAddOrders []*common.Order
		initArrivalDelOrders []*common.Order
		peekTradePair        *common.TradePair
		wantArrivalAddOrders []*common.Order
		wantOrder            *common.Order
	}{
		{
			desc:                 "empty peek",
			initArrivalAddOrders: []*common.Order{},
			initArrivalDelOrders: []*common.Order{},
			peekTradePair:        btc2eth,
			wantArrivalAddOrders: []*common.Order{},
			wantOrder:            nil,
		},
		{
			desc:                 "1 element regular peek",
			initArrivalAddOrders: []*common.Order{mock.Btc2EthOrders[0]},
			initArrivalDelOrders: []*common.Order{},
			peekTradePair:        btc2eth,
			wantArrivalAddOrders: []*common.Order{mock.Btc2EthOrders[0]},
			wantOrder:            mock.Btc2EthOrders[0],
		},
		{
			desc: "4 element regular peek with",
			initArrivalAddOrders: []*common.Order{
				mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Btc2EthOrders[2], mock.Btc2EthOrders[3],
			},
			initArrivalDelOrders: []*common.Order{},
			peekTradePair:        btc2eth,
			wantArrivalAddOrders: []*common.Order{
				mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Btc2EthOrders[2], mock.Btc2EthOrders[3],
			},
			wantOrder: mock.Btc2EthOrders[3],
		},
		{
			desc:                 "1 element peek with 1 unrelated deleted order",
			initArrivalAddOrders: []*common.Order{mock.Btc2EthOrders[0]},
			initArrivalDelOrders: []*common.Order{mock.Btc2EthOrders[1]},
			peekTradePair:        btc2eth,
			wantArrivalAddOrders: []*common.Order{mock.Btc2EthOrders[0]},
			wantOrder:            mock.Btc2EthOrders[0],
		},
		{
			desc:                 "1 element peek with 1 related deleted order",
			initArrivalAddOrders: []*common.Order{mock.Btc2EthOrders[0]},
			initArrivalDelOrders: []*common.Order{mock.Btc2EthOrders[0]},
			peekTradePair:        btc2eth,
			wantArrivalAddOrders: []*common.Order{},
			wantOrder:            nil,
		},
		{
			desc: "4 element peek with first 3 deleted order",
			initArrivalAddOrders: []*common.Order{
				mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Btc2EthOrders[2], mock.Btc2EthOrders[3],
			},
			initArrivalDelOrders: []*common.Order{
				mock.Btc2EthOrders[0], mock.Btc2EthOrders[2], mock.Btc2EthOrders[3],
			},
			peekTradePair:        btc2eth,
			wantArrivalAddOrders: []*common.Order{mock.Btc2EthOrders[1]},
			wantOrder:            mock.Btc2EthOrders[1],
		},
		{
			desc: "4 element peek with first 1 deleted order",
			initArrivalAddOrders: []*common.Order{
				mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Btc2EthOrders[2], mock.Btc2EthOrders[3],
			},
			initArrivalDelOrders: []*common.Order{mock.Btc2EthOrders[3]},
			peekTradePair:        btc2eth,
			wantArrivalAddOrders: []*common.Order{
				mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Btc2EthOrders[2],
			},
			wantOrder: mock.Btc2EthOrders[0],
		},
		{
			desc: "4 element peek with first 2th deleted order",
			initArrivalAddOrders: []*common.Order{
				mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Btc2EthOrders[2], mock.Btc2EthOrders[3],
			},
			initArrivalDelOrders: []*common.Order{mock.Btc2EthOrders[0]},
			peekTradePair:        btc2eth,
			wantArrivalAddOrders: []*common.Order{
				mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Btc2EthOrders[2], mock.Btc2EthOrders[3],
			},
			wantOrder: mock.Btc2EthOrders[3],
		},
	}

	for i, c := range cases {
		orderBook := NewOrderBook(mock.NewMovStore(nil, nil), c.initArrivalAddOrders, c.initArrivalDelOrders)
		gotOrder := orderBook.PeekOrder(c.peekTradePair)
		if !testutil.DeepEqual(gotOrder, c.wantOrder) {
			t.Fatalf("#%d(%s):the key of got order(%v) is not equals key of want order(%v)", i, c.desc, gotOrder, c.wantOrder)
		}

		wantAddOrders, _ := arrangeArrivalAddOrders(c.wantArrivalAddOrders).Load(c.peekTradePair.Key())
		gotAddOrders := orderBook.getArrivalAddOrders(c.peekTradePair.Key())
		if !testutil.DeepEqual(gotAddOrders, wantAddOrders) {
			t.Fatalf("#%d(%s): the got arrivalAddOrders(%v) is differnt than want arrivalAddOrders(%v)", i, c.desc, gotAddOrders, wantAddOrders)
		}
	}
}

func TestAddOrder(t *testing.T) {
	cases := []struct {
		initOrders []*common.Order
		wantOrders []*common.Order
		addOrder   *common.Order
	}{
		{
			initOrders: []*common.Order{},
			addOrder:   &common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 53, RatioDenominator: 1},
			wantOrders: []*common.Order{
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 53, RatioDenominator: 1},
			},
		},
		{
			initOrders: []*common.Order{
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 51, RatioDenominator: 1},
			},
			addOrder: &common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 50, RatioDenominator: 1},
			wantOrders: []*common.Order{
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 51, RatioDenominator: 1},
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 50, RatioDenominator: 1},
			},
		},
		{
			initOrders: []*common.Order{
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 50, RatioDenominator: 1},
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 51, RatioDenominator: 1},
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 52, RatioDenominator: 1},
			},
			addOrder: &common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 53, RatioDenominator: 1},
			wantOrders: []*common.Order{
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 53, RatioDenominator: 1},
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 52, RatioDenominator: 1},
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 51, RatioDenominator: 1},
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 50, RatioDenominator: 1},
			},
		},
		{
			initOrders: []*common.Order{
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 50, RatioDenominator: 1},
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 51, RatioDenominator: 1},
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 52, RatioDenominator: 1},
			},
			addOrder: &common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 49, RatioDenominator: 1},
			wantOrders: []*common.Order{
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 52, RatioDenominator: 1},
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 51, RatioDenominator: 1},
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 50, RatioDenominator: 1},
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 49, RatioDenominator: 1},
			},
		},
		{
			initOrders: []*common.Order{
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 52, RatioDenominator: 1},
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 51, RatioDenominator: 1},
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 49, RatioDenominator: 1},
			},
			addOrder: &common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 50, RatioDenominator: 1},
			wantOrders: []*common.Order{
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 52, RatioDenominator: 1},
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 51, RatioDenominator: 1},
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 50, RatioDenominator: 1},
				&common.Order{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH, RatioNumerator: 49, RatioDenominator: 1},
			},
		},
	}

	for i, c := range cases {
		orderBook := NewOrderBook(mock.NewMovStore(nil, nil), c.initOrders, nil)
		orderBook.AddOrder(c.addOrder)
		if gotOrders := orderBook.getArrivalAddOrders(btc2eth.Key()); !testutil.DeepEqual(gotOrders, c.wantOrders) {
			t.Fatalf("#%d: the gotOrders(%v) is differnt than wantOrders(%v)", i, gotOrders, c.wantOrders)
		}
	}
}
