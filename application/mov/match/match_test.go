package match

import (
	"testing"

	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/application/mov/mock"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/validation"
	"github.com/bytom/vapor/testutil"
)

func TestGenerateMatchedTxs(t *testing.T) {
	btc2eth := &common.TradePair{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH}
	eth2btc := &common.TradePair{FromAssetID: &mock.ETH, ToAssetID: &mock.BTC}
	eth2eos := &common.TradePair{FromAssetID: &mock.ETH, ToAssetID: &mock.EOS}
	eos2btc := &common.TradePair{FromAssetID: &mock.EOS, ToAssetID: &mock.BTC}

	cases := []struct {
		desc            string
		tradePairs      []*common.TradePair
		initStoreOrders []*common.Order
		wantMatchedTxs  []*types.Tx
	}{
		{
			desc:       "full matched",
			tradePairs: []*common.TradePair{btc2eth, eth2btc},
			initStoreOrders: []*common.Order{
				mock.Btc2EthOrders[0], mock.Btc2EthOrders[1],
				mock.Eth2BtcOrders[0],
			},
			wantMatchedTxs: []*types.Tx{
				mock.MatchedTxs[1],
			},
		},
		{
			desc:       "partial matched",
			tradePairs: []*common.TradePair{btc2eth, eth2btc},
			initStoreOrders: []*common.Order{
				mock.Btc2EthOrders[0], mock.Btc2EthOrders[1],
				mock.Eth2BtcOrders[1],
			},
			wantMatchedTxs: []*types.Tx{
				mock.MatchedTxs[0],
			},
		},
		{
			desc:       "partial matched and continue to match",
			tradePairs: []*common.TradePair{btc2eth, eth2btc},
			initStoreOrders: []*common.Order{
				mock.Btc2EthOrders[0], mock.Btc2EthOrders[1],
				mock.Eth2BtcOrders[2],
			},
			wantMatchedTxs: []*types.Tx{
				mock.MatchedTxs[2],
				mock.MatchedTxs[3],
			},
		},
		{
			desc:       "unable to match",
			tradePairs: []*common.TradePair{btc2eth, eth2btc},
			initStoreOrders: []*common.Order{
				mock.Btc2EthOrders[1],
				mock.Eth2BtcOrders[0],
			},
			wantMatchedTxs: []*types.Tx{},
		},
		{
			desc:       "cycle match",
			tradePairs: []*common.TradePair{btc2eth, eth2eos, eos2btc},
			initStoreOrders: []*common.Order{
				mock.Btc2EthOrders[0], mock.Eth2EosOrders[0], mock.Eos2BtcOrders[0],
			},
			wantMatchedTxs: []*types.Tx{
				mock.MatchedTxs[6],
			},
		},
	}

	for i, c := range cases {
		movStore := mock.NewMovStore([]*common.TradePair{btc2eth, eth2btc}, c.initStoreOrders)
		matchEngine := NewEngine(NewOrderBook(movStore, nil, nil), 0.05, mock.NodeProgram)
		var gotMatchedTxs []*types.Tx
		for matchEngine.HasMatchedTx(c.tradePairs...) {
			matchedTx, err := matchEngine.NextMatchedTx(c.tradePairs...)
			if err != nil {
				t.Fatal(err)
			}

			gotMatchedTxs = append(gotMatchedTxs, matchedTx)
		}

		if len(c.wantMatchedTxs) != len(gotMatchedTxs) {
			t.Errorf("#%d(%s) the length of got matched tx is not equals want matched tx", i, c.desc)
			continue
		}

		for i, gotMatchedTx := range gotMatchedTxs {
			if _, err := validation.ValidateTx(gotMatchedTx.Tx, &bc.Block{BlockHeader: &bc.BlockHeader{Version: 1}}); err != nil {
				t.Fatal(err)
			}

			c.wantMatchedTxs[i].Version = 1
			byteData, err := c.wantMatchedTxs[i].MarshalText()
			if err != nil {
				t.Fatal(err)
			}

			c.wantMatchedTxs[i].SerializedSize = uint64(len(byteData))
			wantMatchedTx := types.NewTx(c.wantMatchedTxs[i].TxData)
			if gotMatchedTx.ID != wantMatchedTx.ID {
				t.Errorf("#%d(%s) the tx hash of got matched tx: %s is not equals want matched tx: %s", i, c.desc, gotMatchedTx.ID.String(), wantMatchedTx.ID.String())
			}
		}
	}
}

func TestCalcMatchedTxFee(t *testing.T) {
	cases := []struct {
		desc             string
		tx               *types.TxData
		maxFeeRate       float64
		wantMatchedTxFee map[bc.AssetID]*MatchedTxFee
	}{
		{
			desc:             "fee less than max fee",
			maxFeeRate:       0.05,
			wantMatchedTxFee: map[bc.AssetID]*MatchedTxFee{mock.ETH: {FeeAmount: 10, MaxFeeAmount: 26}},
			tx:               &mock.MatchedTxs[1].TxData,
		},
		{
			desc:             "fee refund in tx",
			maxFeeRate:       0.05,
			wantMatchedTxFee: map[bc.AssetID]*MatchedTxFee{mock.ETH: {FeeAmount: 27, MaxFeeAmount: 27}},
			tx:               &mock.MatchedTxs[2].TxData,
		},
		{
			desc:             "fee is zero",
			maxFeeRate:       0.05,
			wantMatchedTxFee: map[bc.AssetID]*MatchedTxFee{},
			tx:               &mock.MatchedTxs[0].TxData,
		},
	}

	for i, c := range cases {
		gotMatchedTxFee, err := CalcMatchedTxFee(c.tx, c.maxFeeRate)
		if err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(gotMatchedTxFee, c.wantMatchedTxFee) {
			t.Errorf("#%d(%s):fail to caculate matched tx fee, got (%v), want (%v)", i, c.desc, gotMatchedTxFee, c.wantMatchedTxFee)
		}
	}
}

func TestValidateTradePairs(t *testing.T) {
	cases := []struct {
		desc       string
		tradePairs []*common.TradePair
		wantError  bool
	}{
		{
			desc: "valid case of two trade pairs",
			tradePairs: []*common.TradePair{
				{
					FromAssetID: &mock.BTC,
					ToAssetID:   &mock.ETH,
				},
				{
					FromAssetID: &mock.ETH,
					ToAssetID:   &mock.BTC,
				},
			},
			wantError: false,
		},
		{
			desc: "invalid case of two trade pairs",
			tradePairs: []*common.TradePair{
				{
					FromAssetID: &mock.BTC,
					ToAssetID:   &mock.ETH,
				},
				{
					FromAssetID: &mock.ETH,
					ToAssetID:   &mock.EOS,
				},
			},
			wantError: true,
		},
		{
			desc: "valid case of three trade pairs",
			tradePairs: []*common.TradePair{
				{
					FromAssetID: &mock.BTC,
					ToAssetID:   &mock.ETH,
				},
				{
					FromAssetID: &mock.ETH,
					ToAssetID:   &mock.EOS,
				},
				{
					FromAssetID: &mock.EOS,
					ToAssetID:   &mock.BTC,
				},
			},
			wantError: false,
		},
		{
			desc: "invalid case of three trade pairs",
			tradePairs: []*common.TradePair{
				{
					FromAssetID: &mock.BTC,
					ToAssetID:   &mock.ETH,
				},
				{
					FromAssetID: &mock.ETH,
					ToAssetID:   &mock.BTC,
				},
				{
					FromAssetID: &mock.BTC,
					ToAssetID:   &mock.BTC,
				},
			},
			wantError: true,
		},
		{
			desc: "valid case 2 of three trade pairs",
			tradePairs: []*common.TradePair{
				{
					FromAssetID: &mock.BTC,
					ToAssetID:   &mock.ETH,
				},
				{
					FromAssetID: &mock.EOS,
					ToAssetID:   &mock.BTC,
				},
				{
					FromAssetID: &mock.ETH,
					ToAssetID:   &mock.EOS,
				},
			},
			wantError: false,
		},
	}

	for i, c := range cases {
		err := validateTradePairs(c.tradePairs)
		if c.wantError && err == nil {
			t.Errorf("#%d(%s): want error, got no error", i, c.desc)
		}

		if !c.wantError && err != nil {
			t.Errorf("#%d(%s): want no error, got error (%v)", i, c.desc, err)
		}
	}
}

func TestIsMatched(t *testing.T) {
	cases := []struct {
		desc        string
		orders      []*common.Order
		wantMatched bool
	}{
		{
			desc: "ratio is equals",
			orders: []*common.Order{
				{
					FromAssetID:      &mock.BTC,
					ToAssetID:        &mock.ETH,
					RatioNumerator:   6250,
					RatioDenominator: 3,
				},
				{
					FromAssetID:      &mock.ETH,
					ToAssetID:        &mock.BTC,
					RatioNumerator:   3,
					RatioDenominator: 6250,
				},
			},
			wantMatched: true,
		},
		{
			desc: "ratio is equals, and numerator and denominator are multiples of the opposite",
			orders: []*common.Order{
				{
					FromAssetID:      &mock.BTC,
					ToAssetID:        &mock.ETH,
					RatioNumerator:   6250,
					RatioDenominator: 3,
				},
				{
					FromAssetID:      &mock.ETH,
					ToAssetID:        &mock.BTC,
					RatioNumerator:   9,
					RatioDenominator: 18750,
				},
			},
			wantMatched: true,
		},
		{
			desc: "matched with a slight diff",
			orders: []*common.Order{
				{
					FromAssetID:      &mock.BTC,
					ToAssetID:        &mock.ETH,
					RatioNumerator:   62500000000000000,
					RatioDenominator: 30000000000001,
				},
				{
					FromAssetID:      &mock.ETH,
					ToAssetID:        &mock.BTC,
					RatioNumerator:   3,
					RatioDenominator: 6250,
				},
			},
			wantMatched: true,
		},
		{
			desc: "not matched with a slight diff",
			orders: []*common.Order{
				{
					FromAssetID:      &mock.BTC,
					ToAssetID:        &mock.ETH,
					RatioNumerator:   62500000000000001,
					RatioDenominator: 30000000000000,
				},
				{
					FromAssetID:      &mock.ETH,
					ToAssetID:        &mock.BTC,
					RatioNumerator:   3,
					RatioDenominator: 6250,
				},
			},
			wantMatched: false,
		},
		{
			desc: "ring matched",
			orders: []*common.Order{
				{
					FromAssetID:      &mock.BTC,
					ToAssetID:        &mock.ETH,
					RatioNumerator:   2000000003,
					RatioDenominator: 100000001,
				},
				{
					FromAssetID:      &mock.ETH,
					ToAssetID:        &mock.EOS,
					RatioNumerator:   400000000001,
					RatioDenominator: 2000000003,
				},
				{
					FromAssetID:      &mock.EOS,
					ToAssetID:        &mock.BTC,
					RatioNumerator:   100000001,
					RatioDenominator: 400000000001,
				},
			},
			wantMatched: true,
		},
		{
			desc: "ring matched with a slight diff",
			orders: []*common.Order{
				{
					FromAssetID:      &mock.BTC,
					ToAssetID:        &mock.ETH,
					RatioNumerator:   2000000003,
					RatioDenominator: 100000001,
				},
				{
					FromAssetID:      &mock.ETH,
					ToAssetID:        &mock.EOS,
					RatioNumerator:   400000000001,
					RatioDenominator: 2000000003,
				},
				{
					FromAssetID:      &mock.EOS,
					ToAssetID:        &mock.BTC,
					RatioNumerator:   100000000,
					RatioDenominator: 400000000001,
				},
			},
			wantMatched: true,
		},
		{
			desc: "ring fail matched with a slight diff",
			orders: []*common.Order{
				{
					FromAssetID:      &mock.BTC,
					ToAssetID:        &mock.ETH,
					RatioNumerator:   2000000003,
					RatioDenominator: 100000001,
				},
				{
					FromAssetID:      &mock.ETH,
					ToAssetID:        &mock.EOS,
					RatioNumerator:   400000000001,
					RatioDenominator: 2000000003,
				},
				{
					FromAssetID:      &mock.EOS,
					ToAssetID:        &mock.BTC,
					RatioNumerator:   100000002,
					RatioDenominator: 400000000001,
				},
			},
			wantMatched: false,
		},
	}

	for i, c := range cases {
		gotMatched := IsMatched(c.orders)
		if gotMatched != c.wantMatched {
			t.Errorf("#%d(%s) got matched:%v, want matched:%v", i, c.desc, gotMatched, c.wantMatched)
		}
	}
}
