package mock

import (
	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/vm"
	"github.com/bytom/vapor/protocol/vm/vmutil"
	"github.com/bytom/vapor/testutil"
)

var (
	BTC         = bc.NewAssetID([32]byte{1})
	ETH         = bc.NewAssetID([32]byte{2})
	NodeProgram = []byte{0x58}

	Btc2EthOrders = []*common.Order{
		{
			FromAssetID: &BTC,
			ToAssetID:   &ETH,
			Rate:        50,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("37b8edf656e45a7addf47f5626e114a8c394d918a36f61b5a2905675a09b40ae")),
				SourcePos:      0,
				Amount:         10,
				ControlProgram: MustCreateP2WMCProgram(ETH, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251"), 50, 1),
			},
		},
		{
			FromAssetID: &BTC,
			ToAssetID:   &ETH,
			Rate:        53,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("3ec2bbfb499a8736d377b547eee5392bcddf7ec2b287e9ed20b5938c3d84e7cd")),
				SourcePos:      0,
				Amount:         20,
				ControlProgram: MustCreateP2WMCProgram(ETH, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19252"), 53, 1),
			},
		},
		{
			FromAssetID: &BTC,
			ToAssetID:   &ETH,
			Rate:        52,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("1232bbfb499a8736d377b547eee5392bcddf7ec2b287e9ed20b5938c3d84e7cd")),
				SourcePos:      0,
				Amount:         15,
				ControlProgram: MustCreateP2WMCProgram(ETH, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19252"), 53, 1),
			},
		},
		{
			FromAssetID: &BTC,
			ToAssetID:   &ETH,
			Rate:        49,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("7872bbfb499a8736d377b547eee5392bcddf7ec2b287e9ed20b5938c3d84e7cd")),
				SourcePos:      0,
				Amount:         17,
				ControlProgram: MustCreateP2WMCProgram(ETH, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19252"), 53, 1),
			},
		},
	}

	Eth2BtcOrders = []*common.Order{
		{
			FromAssetID: &ETH,
			ToAssetID:   &BTC,
			Rate:        1 / 51.0,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("fba43ff5155209cb1769e2ec0e1d4a33accf899c740865edfc6d1de39b873b29")),
				SourcePos:      0,
				Amount:         510,
				ControlProgram: MustCreateP2WMCProgram(BTC, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19253"), 1, 51.0),
			},
		},
		{
			FromAssetID: &ETH,
			ToAssetID:   &BTC,
			Rate:        1 / 52.0,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("05f24bb847db823075d81786aa270748e02602199cd009c0284f928503846a5a")),
				SourcePos:      0,
				Amount:         416,
				ControlProgram: MustCreateP2WMCProgram(BTC, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19254"), 1, 52.0),
			},
		},
		{
			FromAssetID: &ETH,
			ToAssetID:   &BTC,
			Rate:        1 / 54.0,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("119a02980796dc352cf6475457463aef5666f66622088de3551fa73a65f0d201")),
				SourcePos:      0,
				Amount:         810,
				ControlProgram: MustCreateP2WMCProgram(BTC, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19255"), 1, 54.0),
			},
		},
	}

	Btc2EthMakerTxs = []*types.Tx{
		// Btc2EthOrders[0]
		types.NewTx(types.TxData{
			Inputs:  []*types.TxInput{types.NewSpendInput(nil, *Btc2EthOrders[0].Utxo.SourceID, *Btc2EthOrders[0].FromAssetID, Btc2EthOrders[0].Utxo.Amount, Btc2EthOrders[0].Utxo.SourcePos, []byte{0x51})},
			Outputs: []*types.TxOutput{types.NewIntraChainOutput(*Btc2EthOrders[0].FromAssetID, Btc2EthOrders[0].Utxo.Amount, Btc2EthOrders[0].Utxo.ControlProgram)},
		}),
		// Btc2EthOrders[1]
		types.NewTx(types.TxData{
			Inputs:  []*types.TxInput{types.NewSpendInput(nil, *Btc2EthOrders[1].Utxo.SourceID, *Btc2EthOrders[1].FromAssetID, Btc2EthOrders[1].Utxo.Amount, Btc2EthOrders[1].Utxo.SourcePos, []byte{0x51})},
			Outputs: []*types.TxOutput{types.NewIntraChainOutput(*Btc2EthOrders[1].FromAssetID, Btc2EthOrders[1].Utxo.Amount, Btc2EthOrders[1].Utxo.ControlProgram)},
		}),
		// Btc2EthOrders[2]
		types.NewTx(types.TxData{
			Inputs:  []*types.TxInput{types.NewSpendInput(nil, *Btc2EthOrders[2].Utxo.SourceID, *Btc2EthOrders[2].FromAssetID, Btc2EthOrders[2].Utxo.Amount, Btc2EthOrders[2].Utxo.SourcePos, []byte{0x51})},
			Outputs: []*types.TxOutput{types.NewIntraChainOutput(*Btc2EthOrders[2].FromAssetID, Btc2EthOrders[2].Utxo.Amount, Btc2EthOrders[2].Utxo.ControlProgram)},
		}),
		// Btc2EthOrders[3]
		types.NewTx(types.TxData{
			Inputs:  []*types.TxInput{types.NewSpendInput(nil, *Btc2EthOrders[3].Utxo.SourceID, *Btc2EthOrders[3].FromAssetID, Btc2EthOrders[3].Utxo.Amount, Btc2EthOrders[3].Utxo.SourcePos, []byte{0x51})},
			Outputs: []*types.TxOutput{types.NewIntraChainOutput(*Btc2EthOrders[3].FromAssetID, Btc2EthOrders[3].Utxo.Amount, Btc2EthOrders[3].Utxo.ControlProgram)},
		}),
	}

	Eth2BtcMakerTxs = []*types.Tx{
		// Eth2Btc[0]
		types.NewTx(types.TxData{
			Inputs:  []*types.TxInput{types.NewSpendInput(nil, *Eth2BtcOrders[0].Utxo.SourceID, *Eth2BtcOrders[0].FromAssetID, Eth2BtcOrders[0].Utxo.Amount, Eth2BtcOrders[0].Utxo.SourcePos, []byte{0x51})},
			Outputs: []*types.TxOutput{types.NewIntraChainOutput(*Eth2BtcOrders[0].FromAssetID, Eth2BtcOrders[0].Utxo.Amount, Eth2BtcOrders[0].Utxo.ControlProgram)},
		}),
		// Eth2Btc[1]
		types.NewTx(types.TxData{
			Inputs:  []*types.TxInput{types.NewSpendInput(nil, *Eth2BtcOrders[1].Utxo.SourceID, *Eth2BtcOrders[1].FromAssetID, Eth2BtcOrders[1].Utxo.Amount, Eth2BtcOrders[1].Utxo.SourcePos, []byte{0x51})},
			Outputs: []*types.TxOutput{types.NewIntraChainOutput(*Eth2BtcOrders[1].FromAssetID, Eth2BtcOrders[1].Utxo.Amount, Eth2BtcOrders[1].Utxo.ControlProgram)},
		}),
		// Eth2Btc[2]
		types.NewTx(types.TxData{
			Inputs:  []*types.TxInput{types.NewSpendInput(nil, *Eth2BtcOrders[2].Utxo.SourceID, *Eth2BtcOrders[2].FromAssetID, Eth2BtcOrders[2].Utxo.Amount, Eth2BtcOrders[2].Utxo.SourcePos, []byte{0x51})},
			Outputs: []*types.TxOutput{types.NewIntraChainOutput(*Eth2BtcOrders[2].FromAssetID, Eth2BtcOrders[2].Utxo.Amount, Eth2BtcOrders[2].Utxo.ControlProgram)},
		}),
	}

	MatchedTxs = []*types.Tx{
		// partial matched transaction from Btc2EthOrders[0], Eth2BtcOrders[1]
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{vm.Int64Bytes(416), vm.Int64Bytes(0), vm.Int64Bytes(0)}, *Btc2EthOrders[0].Utxo.SourceID, *Btc2EthOrders[0].FromAssetID, Btc2EthOrders[0].Utxo.Amount, Btc2EthOrders[0].Utxo.SourcePos, Btc2EthOrders[0].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(2), vm.Int64Bytes(1)}, *Eth2BtcOrders[1].Utxo.SourceID, *Eth2BtcOrders[1].FromAssetID, Eth2BtcOrders[1].Utxo.Amount, Eth2BtcOrders[1].Utxo.SourcePos, Eth2BtcOrders[1].Utxo.ControlProgram),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 416, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
				// re-order
				types.NewIntraChainOutput(*Btc2EthOrders[0].FromAssetID, 2, Btc2EthOrders[0].Utxo.ControlProgram),
				types.NewIntraChainOutput(*Eth2BtcOrders[1].ToAssetID, 8, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19254")),
			},
		}),

		// full matched transaction from Btc2EthOrders[0], Eth2BtcOrders[0]
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *Btc2EthOrders[0].Utxo.SourceID, *Btc2EthOrders[0].FromAssetID, Btc2EthOrders[0].Utxo.Amount, Btc2EthOrders[0].Utxo.SourcePos, Btc2EthOrders[0].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(1), vm.Int64Bytes(1)}, *Eth2BtcOrders[0].Utxo.SourceID, *Eth2BtcOrders[0].FromAssetID, Eth2BtcOrders[0].Utxo.Amount, Eth2BtcOrders[0].Utxo.SourcePos, Eth2BtcOrders[0].Utxo.ControlProgram),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 500, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
				types.NewIntraChainOutput(*Eth2BtcOrders[0].ToAssetID, 10, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19253")),
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 10, NodeProgram),
			},
		}),

		// partial matched transaction from Btc2EthOrders[0], Eth2BtcOrders[2]
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *Btc2EthOrders[0].Utxo.SourceID, *Btc2EthOrders[0].FromAssetID, Btc2EthOrders[0].Utxo.Amount, Btc2EthOrders[0].Utxo.SourcePos, Btc2EthOrders[0].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(10), vm.Int64Bytes(1), vm.Int64Bytes(0)}, *Eth2BtcOrders[2].Utxo.SourceID, *Eth2BtcOrders[2].FromAssetID, Eth2BtcOrders[2].Utxo.Amount, Eth2BtcOrders[2].Utxo.SourcePos, Eth2BtcOrders[2].Utxo.ControlProgram),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 500, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
				types.NewIntraChainOutput(*Eth2BtcOrders[2].ToAssetID, 10, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19255")),
				// re-order
				types.NewIntraChainOutput(*Eth2BtcOrders[2].FromAssetID, 270, Eth2BtcOrders[2].Utxo.ControlProgram),
				// fee
				types.NewIntraChainOutput(*Eth2BtcOrders[2].FromAssetID, 27, NodeProgram),
				// refund
				types.NewIntraChainOutput(*Eth2BtcOrders[2].FromAssetID, 6, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
				types.NewIntraChainOutput(*Eth2BtcOrders[2].FromAssetID, 7, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19255")),
			},
		}),
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(0)}, *Btc2EthOrders[1].Utxo.SourceID, *Btc2EthOrders[1].FromAssetID, Btc2EthOrders[1].Utxo.Amount, Btc2EthOrders[1].Utxo.SourcePos, Btc2EthOrders[1].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(2), vm.Int64Bytes(1)}, testutil.MustDecodeHash("39bdb7058a0c31fb740af8e3c382bf608efff1b041cd4dd461332722ad24552a"), *Eth2BtcOrders[2].FromAssetID, 270, 2, Eth2BtcOrders[2].Utxo.ControlProgram),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(*Btc2EthOrders[1].ToAssetID, 270, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19252")),
				// re-order
				types.NewIntraChainOutput(*Btc2EthOrders[1].FromAssetID, 15, Btc2EthOrders[1].Utxo.ControlProgram),
				types.NewIntraChainOutput(*Eth2BtcOrders[2].ToAssetID, 5, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19255")),
			},
		}),

		// partial matched transaction from Btc2EthMakerTxs[0], Eth2BtcMakerTxs[1]
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *MustNewOrderFromOutput(Btc2EthMakerTxs[0], 0).Utxo.SourceID, *Btc2EthOrders[0].FromAssetID, Btc2EthOrders[0].Utxo.Amount, 0, Btc2EthOrders[0].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(1), vm.Int64Bytes(1)}, *MustNewOrderFromOutput(Eth2BtcMakerTxs[1], 0).Utxo.SourceID, *Eth2BtcOrders[1].FromAssetID, Eth2BtcOrders[1].Utxo.Amount, 0, Eth2BtcOrders[1].Utxo.ControlProgram),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 416, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
				// re-order
				types.NewIntraChainOutput(*Btc2EthOrders[0].FromAssetID, 2, Btc2EthOrders[0].Utxo.ControlProgram),
				types.NewIntraChainOutput(*Eth2BtcOrders[1].ToAssetID, 8, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19254")),
			},
		}),
	}
)

func MustCreateP2WMCProgram(requestAsset bc.AssetID, sellerProgram []byte, ratioNumerator, ratioDenominator int64) []byte {
	contractArgs := vmutil.MagneticContractArgs{
		RequestedAsset:   requestAsset,
		RatioNumerator:   ratioNumerator,
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

func MustNewOrderFromOutput(tx *types.Tx, outputIndex int) *common.Order {
	order, err := common.NewOrderFromOutput(tx, outputIndex)
	if err != nil {
		panic(err)
	}

	return order
}

func hashPtr(hash bc.Hash) *bc.Hash {
	return &hash
}
