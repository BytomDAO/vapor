package match

import (
	"testing"

	"github.com/vapor/application/mov/common"
	"github.com/vapor/application/mov/mock"
	"github.com/vapor/protocol/bc/types"
)

func TestGenerateMatchedTxs(t *testing.T) {
	btc2eth := &common.TradePair{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH}
	eth2btc := &common.TradePair{FromAssetID: &mock.ETH, ToAssetID: &mock.BTC}

	cases := []struct {
		desc            string
		tradePair       *common.TradePair
		initStoreOrders []*common.Order
		wantMatchedTxs  []*types.Tx
	}{
		{
			desc:      "full matched",
			tradePair: &common.TradePair{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH},
			initStoreOrders: []*common.Order{
				mock.Btc2EthOrders[0], mock.Btc2EthOrders[1],
				mock.Eth2BtcOrders[0],
			},
			wantMatchedTxs: []*types.Tx{
				mock.MatchedTxs[1],
			},
		},
		{
			desc:      "partial matched",
			tradePair: &common.TradePair{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH},
			initStoreOrders: []*common.Order{
				mock.Btc2EthOrders[0], mock.Btc2EthOrders[1],
				mock.Eth2BtcOrders[1],
			},
			wantMatchedTxs: []*types.Tx{
				mock.MatchedTxs[0],
			},
		},
		{
			desc:      "partial matched and continue to match",
			tradePair: &common.TradePair{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH},
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
			desc:      "unable to match",
			tradePair: &common.TradePair{FromAssetID: &mock.BTC, ToAssetID: &mock.ETH},
			initStoreOrders: []*common.Order{
				mock.Btc2EthOrders[1],
				mock.Eth2BtcOrders[0],
			},
			wantMatchedTxs: []*types.Tx{},
		},
	}

	for i, c := range cases {
		movStore := mock.NewMovStore([]*common.TradePair{btc2eth, eth2btc}, c.initStoreOrders)
		matchEngine := NewEngine(NewOrderTable(movStore, nil, nil), 0.05, []byte{0x51})
		var gotMatchedTxs []*types.Tx
		for matchEngine.HasMatchedTx(c.tradePair, c.tradePair.Reverse()) {
			matchedTx, err := matchEngine.NextMatchedTx(c.tradePair, c.tradePair.Reverse())
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
