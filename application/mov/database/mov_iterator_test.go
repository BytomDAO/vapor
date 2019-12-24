package database

import (
	"testing"

	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/application/mov/mock"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/testutil"
)

var (
	asset1 = bc.NewAssetID([32]byte{1})
	asset2 = bc.NewAssetID([32]byte{2})
	asset3 = bc.NewAssetID([32]byte{3})
	asset4 = bc.NewAssetID([32]byte{4})

	order1 = &common.Order{FromAssetID: assetID1, ToAssetID: assetID2, RatioNumerator: 1, RatioDenominator: 10}
	order2 = &common.Order{FromAssetID: assetID1, ToAssetID: assetID2, RatioNumerator: 2, RatioDenominator: 10}
	order3 = &common.Order{FromAssetID: assetID1, ToAssetID: assetID2, RatioNumerator: 3, RatioDenominator: 10}
	order4 = &common.Order{FromAssetID: assetID1, ToAssetID: assetID2, RatioNumerator: 4, RatioDenominator: 10}
	order5 = &common.Order{FromAssetID: assetID1, ToAssetID: assetID2, RatioNumerator: 5, RatioDenominator: 10}
)

func TestTradePairIterator(t *testing.T) {
	cases := []struct {
		desc            string
		storeTradePairs []*common.TradePair
		wantTradePairs  []*common.TradePair
	}{
		{
			desc: "normal case",
			storeTradePairs: []*common.TradePair{
				{
					FromAssetID: &asset1,
					ToAssetID:   &asset2,
				},
				{
					FromAssetID: &asset1,
					ToAssetID:   &asset3,
				},
				{
					FromAssetID: &asset1,
					ToAssetID:   &asset4,
				},
			},
			wantTradePairs: []*common.TradePair{
				{
					FromAssetID: &asset1,
					ToAssetID:   &asset2,
				},
				{
					FromAssetID: &asset1,
					ToAssetID:   &asset3,
				},
				{
					FromAssetID: &asset1,
					ToAssetID:   &asset4,
				},
			},
		},
		{
			desc: "num of trade pairs more than one return",
			storeTradePairs: []*common.TradePair{
				{
					FromAssetID: &asset1,
					ToAssetID:   &asset2,
				},
				{
					FromAssetID: &asset1,
					ToAssetID:   &asset3,
				},
				{
					FromAssetID: &asset1,
					ToAssetID:   &asset4,
				},
				{
					FromAssetID: &asset2,
					ToAssetID:   &asset1,
				},
			},
			wantTradePairs: []*common.TradePair{
				{
					FromAssetID: &asset1,
					ToAssetID:   &asset2,
				},
				{
					FromAssetID: &asset1,
					ToAssetID:   &asset3,
				},
				{
					FromAssetID: &asset1,
					ToAssetID:   &asset4,
				},
				{
					FromAssetID: &asset2,
					ToAssetID:   &asset1,
				},
			},
		},
		{
			desc:            "store is empty",
			storeTradePairs: []*common.TradePair{},
			wantTradePairs:  []*common.TradePair{},
		},
	}

	for i, c := range cases {
		store := mock.NewMovStore(c.storeTradePairs, nil)
		var gotTradePairs []*common.TradePair
		iterator := NewTradePairIterator(store)
		for iterator.HasNext() {
			gotTradePairs = append(gotTradePairs, iterator.Next())
		}
		if !testutil.DeepEqual(c.wantTradePairs, gotTradePairs) {
			t.Errorf("#%d(%s):got trade pairs is not equals want trade pairs", i, c.desc)
		}
	}
}

func TestOrderIterator(t *testing.T) {
	cases := []struct {
		desc        string
		tradePair   *common.TradePair
		storeOrders []*common.Order
		wantOrders  []*common.Order
	}{
		{
			desc:        "normal case",
			tradePair:   &common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2},
			storeOrders: []*common.Order{order1, order2, order3},
			wantOrders:  []*common.Order{order1, order2, order3},
		},
		{
			desc:        "num of orders more than one return",
			tradePair:   &common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2},
			storeOrders: []*common.Order{order1, order2, order3, order4, order5},
			wantOrders:  []*common.Order{order1, order2, order3, order4, order5},
		},
		{
			desc:        "only one order",
			tradePair:   &common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2},
			storeOrders: []*common.Order{order1},
			wantOrders:  []*common.Order{order1},
		},
		{
			desc:        "store is empty",
			tradePair:   &common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2},
			storeOrders: []*common.Order{},
			wantOrders:  []*common.Order{},
		},
	}

	for i, c := range cases {
		store := mock.NewMovStore(nil, c.storeOrders)

		var gotOrders []*common.Order
		iterator := NewOrderIterator(store, c.tradePair)
		for iterator.HasNext() {
			gotOrders = append(gotOrders, iterator.NextBatch()...)
		}
		if !testutil.DeepEqual(c.wantOrders, gotOrders) {
			t.Errorf("#%d(%s):got orders it not equals want orders", i, c.desc)
		}
	}
}
