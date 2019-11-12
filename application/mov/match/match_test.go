package match

import (
	"testing"

	"github.com/vapor/protocol/vm"
	"github.com/vapor/testutil"

	"github.com/vapor/application/mov/common"
	"github.com/vapor/application/mov/database"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm/vmutil"
)

var (
	btc = bc.NewAssetID([32]byte{1})
	eth = bc.NewAssetID([32]byte{2})

	orders = []*common.Order{
		// btc -> eth
		{
			FromAssetID: &btc,
			ToAssetID:   &eth,
			Rate:        50,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("37b8edf656e45a7addf47f5626e114a8c394d918a36f61b5a2905675a09b40ae")),
				SourcePos:      0,
				Amount:         10,
				ControlProgram: mustCreateP2WMCProgram(eth, testutil.MustDecodeHexString("51"), 50, 1),
			},
		},
		{
			FromAssetID: &btc,
			ToAssetID:   &eth,
			Rate:        53,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("3ec2bbfb499a8736d377b547eee5392bcddf7ec2b287e9ed20b5938c3d84e7cd")),
				SourcePos:      0,
				Amount:         20,
				ControlProgram: mustCreateP2WMCProgram(eth, testutil.MustDecodeHexString("52"), 53, 1),
			},
		},

		// eth -> btc
		{
			FromAssetID: &eth,
			ToAssetID:   &btc,
			Rate:        1 / 51.0,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("fba43ff5155209cb1769e2ec0e1d4a33accf899c740865edfc6d1de39b873b29")),
				SourcePos:      0,
				Amount:         510,
				ControlProgram: mustCreateP2WMCProgram(btc, testutil.MustDecodeHexString("53"), 1, 51.0),
			},
		},
		{
			FromAssetID: &eth,
			ToAssetID:   &btc,
			Rate:        1 / 52.0,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("05f24bb847db823075d81786aa270748e02602199cd009c0284f928503846a5a")),
				SourcePos:      0,
				Amount:         416,
				ControlProgram: mustCreateP2WMCProgram(btc, testutil.MustDecodeHexString("54"), 1, 52.0),
			},
		},
		{
			FromAssetID: &eth,
			ToAssetID:   &btc,
			Rate:        1 / 54.0,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("119a02980796dc352cf6475457463aef5666f66622088de3551fa73a65f0d201")),
				SourcePos:      0,
				Amount:         810,
				ControlProgram: mustCreateP2WMCProgram(btc, testutil.MustDecodeHexString("55"), 1, 54.0),
			},
		},
	}
)

func TestGenerateMatchedTxs(t *testing.T) {
	btc2eth := &common.TradePair{FromAssetID: &btc, ToAssetID: &eth}
	eth2btc := &common.TradePair{FromAssetID: &eth, ToAssetID: &btc}

	cases := []struct {
		desc           string
		tradePair      *common.TradePair
		storeOrderMap  map[string][]*common.Order
		wantMatchedTxs []*types.TxData
	}{
		{
			desc:      "full matched",
			tradePair: &common.TradePair{FromAssetID: &btc, ToAssetID: &eth},
			storeOrderMap: map[string][]*common.Order{
				btc2eth.Key(): {orders[0], orders[1]},
				eth2btc.Key(): {orders[2], orders[3]},
			},
			wantMatchedTxs: []*types.TxData{
				{
					Inputs: []*types.TxInput{
						types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *orders[0].Utxo.SourceID, *orders[0].FromAssetID, orders[0].Utxo.Amount, orders[0].Utxo.SourcePos, orders[0].Utxo.ControlProgram),
						types.NewSpendInput([][]byte{vm.Int64Bytes(1), vm.Int64Bytes(1)}, *orders[2].Utxo.SourceID, *orders[2].FromAssetID, orders[2].Utxo.Amount, orders[2].Utxo.SourcePos, orders[2].Utxo.ControlProgram),
					},
					Outputs: []*types.TxOutput{
						types.NewIntraChainOutput(*orders[0].ToAssetID, 500, testutil.MustDecodeHexString("51")),
						types.NewIntraChainOutput(*orders[2].ToAssetID, 10, testutil.MustDecodeHexString("53")),
						types.NewIntraChainOutput(*orders[0].ToAssetID, 10, []byte{0x51}),
					},
				},
			},
		},
		{
			desc:      "partial matched",
			tradePair: &common.TradePair{FromAssetID: &btc, ToAssetID: &eth},
			storeOrderMap: map[string][]*common.Order{
				btc2eth.Key(): {orders[0], orders[1]},
				eth2btc.Key(): {orders[3]},
			},
			wantMatchedTxs: []*types.TxData{
				{
					Inputs: []*types.TxInput{
						types.NewSpendInput([][]byte{vm.Int64Bytes(416), vm.Int64Bytes(0), vm.Int64Bytes(0)}, *orders[0].Utxo.SourceID, *orders[0].FromAssetID, orders[0].Utxo.Amount, orders[0].Utxo.SourcePos, orders[0].Utxo.ControlProgram),
						types.NewSpendInput([][]byte{vm.Int64Bytes(2), vm.Int64Bytes(1)}, *orders[3].Utxo.SourceID, *orders[3].FromAssetID, orders[3].Utxo.Amount, orders[3].Utxo.SourcePos, orders[3].Utxo.ControlProgram),
					},
					Outputs: []*types.TxOutput{
						types.NewIntraChainOutput(*orders[0].ToAssetID, 416, testutil.MustDecodeHexString("51")),
						// re-order
						types.NewIntraChainOutput(*orders[0].FromAssetID, 2, orders[0].Utxo.ControlProgram),
						types.NewIntraChainOutput(*orders[3].ToAssetID, 8, testutil.MustDecodeHexString("54")),
					},
				},
			},
		},
		{
			desc:      "partial matched and continue to match",
			tradePair: &common.TradePair{FromAssetID: &btc, ToAssetID: &eth},
			storeOrderMap: map[string][]*common.Order{
				btc2eth.Key(): {orders[0], orders[1]},
				eth2btc.Key(): {orders[4]},
			},
			wantMatchedTxs: []*types.TxData{
				{
					Inputs: []*types.TxInput{
						types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *orders[0].Utxo.SourceID, *orders[0].FromAssetID, orders[0].Utxo.Amount, orders[0].Utxo.SourcePos, orders[0].Utxo.ControlProgram),
						types.NewSpendInput([][]byte{vm.Int64Bytes(10), vm.Int64Bytes(1), vm.Int64Bytes(0)}, *orders[4].Utxo.SourceID, *orders[4].FromAssetID, orders[4].Utxo.Amount, orders[4].Utxo.SourcePos, orders[4].Utxo.ControlProgram),
					},
					Outputs: []*types.TxOutput{
						types.NewIntraChainOutput(*orders[0].ToAssetID, 500, testutil.MustDecodeHexString("51")),
						types.NewIntraChainOutput(*orders[4].ToAssetID, 10, testutil.MustDecodeHexString("55")),
						// re-order
						types.NewIntraChainOutput(*orders[4].FromAssetID, 270, orders[4].Utxo.ControlProgram),
						// fee
						types.NewIntraChainOutput(*orders[4].FromAssetID, 27, []byte{0x51}),
						// refund
						types.NewIntraChainOutput(*orders[4].FromAssetID, 6, testutil.MustDecodeHexString("51")),
						types.NewIntraChainOutput(*orders[4].FromAssetID, 7, testutil.MustDecodeHexString("55")),
					},
				},
				{
					Inputs: []*types.TxInput{
						types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(0)}, *orders[1].Utxo.SourceID, *orders[1].FromAssetID, orders[1].Utxo.Amount, orders[1].Utxo.SourcePos, orders[1].Utxo.ControlProgram),
						types.NewSpendInput([][]byte{vm.Int64Bytes(2), vm.Int64Bytes(1)}, testutil.MustDecodeHash("f47177c12d25f5316eb377ea006e77bf07e4f9646860e4641e313e004f9aa989"), *orders[4].FromAssetID, 270, 2, orders[4].Utxo.ControlProgram),
					},
					Outputs: []*types.TxOutput{
						types.NewIntraChainOutput(*orders[1].ToAssetID, 270, testutil.MustDecodeHexString("52")),
						// re-order
						types.NewIntraChainOutput(*orders[1].FromAssetID, 15, orders[1].Utxo.ControlProgram),
						types.NewIntraChainOutput(*orders[4].ToAssetID, 5, testutil.MustDecodeHexString("55")),
					},
				},
			},
		},
	}

	for i, c := range cases {
		movStore := &database.MockMovStore{OrderMap: c.storeOrderMap}
		matchEngine := NewEngine(movStore, 0.05, []byte{0x51})
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
			wantMatchedTx := types.NewTx(*c.wantMatchedTxs[i])
			if gotMatchedTx.ID != wantMatchedTx.ID {
				t.Errorf("#%d(%s) the tx hash of got matched tx: %s is not equals want matched tx: %s", i, c.desc, gotMatchedTx.ID.String(), wantMatchedTx.ID.String())
			}
		}
	}
}

func mustCreateP2WMCProgram(requestAsset bc.AssetID, sellerProgram []byte, ratioMolecule, ratioDenominator int64) []byte {
	contractArgs := vmutil.MagneticContractArgs{
		RequestedAsset:   requestAsset,
		RatioNumerator:   ratioMolecule,
		RatioDenominator: ratioDenominator,
		SellerProgram:    sellerProgram,
		SellerKey:        testutil.MustDecodeHexString("ad79ec6bd3a6d6dbe4d0ee902afc99a12b9702fb63edce5f651db3081d868b75"),
	}
	program, err := vmutil.P2WMCProgram(contractArgs)
	if err != nil {
		panic(err)
	}
	return program
}

func hashPtr(hash bc.Hash) *bc.Hash {
	return &hash
}
