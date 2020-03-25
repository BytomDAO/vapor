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
	BTC           = bc.NewAssetID([32]byte{1})
	ETH           = bc.NewAssetID([32]byte{2})
	EOS           = bc.NewAssetID([32]byte{3})
	ETC           = bc.NewAssetID([32]byte{4})
	RewardProgram = []byte{0x58}

	Btc2EthOrders = []*common.Order{
		{
			FromAssetID:      &BTC,
			ToAssetID:        &ETH,
			RatioNumerator:   50,
			RatioDenominator: 1,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("37b8edf656e45a7addf47f5626e114a8c394d918a36f61b5a2905675a09b40ae")),
				SourcePos:      0,
				Amount:         10,
				ControlProgram: MustCreateP2WMCProgram(ETH, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251"), 50, 1),
			},
		},
		{
			FromAssetID:      &BTC,
			ToAssetID:        &ETH,
			RatioNumerator:   53,
			RatioDenominator: 1,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("3ec2bbfb499a8736d377b547eee5392bcddf7ec2b287e9ed20b5938c3d84e7cd")),
				SourcePos:      0,
				Amount:         20,
				ControlProgram: MustCreateP2WMCProgram(ETH, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19252"), 53, 1),
			},
		},
		{
			FromAssetID:      &BTC,
			ToAssetID:        &ETH,
			RatioNumerator:   52,
			RatioDenominator: 1,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("1232bbfb499a8736d377b547eee5392bcddf7ec2b287e9ed20b5938c3d84e7cd")),
				SourcePos:      0,
				Amount:         15,
				ControlProgram: MustCreateP2WMCProgram(ETH, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19252"), 53, 1),
			},
		},
		{
			FromAssetID:      &BTC,
			ToAssetID:        &ETH,
			RatioNumerator:   49,
			RatioDenominator: 1,
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
			FromAssetID:      &ETH,
			ToAssetID:        &BTC,
			RatioNumerator:   1,
			RatioDenominator: 51,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("fba43ff5155209cb1769e2ec0e1d4a33accf899c740865edfc6d1de39b873b29")),
				SourcePos:      0,
				Amount:         510,
				ControlProgram: MustCreateP2WMCProgram(BTC, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19253"), 1, 51.0),
			},
		},
		{
			FromAssetID:      &ETH,
			ToAssetID:        &BTC,
			RatioNumerator:   1,
			RatioDenominator: 52,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("05f24bb847db823075d81786aa270748e02602199cd009c0284f928503846a5a")),
				SourcePos:      0,
				Amount:         416,
				ControlProgram: MustCreateP2WMCProgram(BTC, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19254"), 1, 52.0),
			},
		},
		{
			FromAssetID:      &ETH,
			ToAssetID:        &BTC,
			RatioNumerator:   1,
			RatioDenominator: 54,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("119a02980796dc352cf6475457463aef5666f66622088de3551fa73a65f0d201")),
				SourcePos:      0,
				Amount:         810,
				ControlProgram: MustCreateP2WMCProgram(BTC, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19255"), 1, 54.0),
			},
		},
		{
			FromAssetID:      &ETH,
			ToAssetID:        &BTC,
			RatioNumerator:   1,
			RatioDenominator: 150,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("82752cda63c877a8529d7a7461da6096673e45b3e0b019ce44aa18687ad20445")),
				SourcePos:      0,
				Amount:         600,
				ControlProgram: MustCreateP2WMCProgram(BTC, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19256"), 1, 150.0),
			},
		},
	}

	Eos2EtcOrders = []*common.Order{
		{
			FromAssetID:      &EOS,
			ToAssetID:        &ETC,
			RatioNumerator:   1,
			RatioDenominator: 2,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("119a02980796dc352cf6475457463aef5666f66622088de3551fa73a65f0d202")),
				SourcePos:      0,
				Amount:         100,
				ControlProgram: MustCreateP2WMCProgram(ETC, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19255"), 1, 2.0),
			},
		},
	}

	Etc2EosOrders = []*common.Order{
		{
			FromAssetID:      &ETC,
			ToAssetID:        &EOS,
			RatioNumerator:   2,
			RatioDenominator: 1,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("119a02980796dc352cf6475457463aef5666f66622088de3551fa73a65f0d203")),
				SourcePos:      0,
				Amount:         50,
				ControlProgram: MustCreateP2WMCProgram(EOS, testutil.MustDecodeHexString("0014df7a97e53bbe278e4e44810b0a760fb472daa9a3"), 2, 1.0),
			},
		},
	}

	Eth2EosOrders = []*common.Order{
		{
			FromAssetID:      &ETH,
			ToAssetID:        &EOS,
			RatioNumerator:   2,
			RatioDenominator: 1,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("c1502d03946e4ea92abdb33f51638b181839bd0d8767acc2ee5c665b659c4b13")),
				SourcePos:      0,
				Amount:         500,
				ControlProgram: MustCreateP2WMCProgram(EOS, testutil.MustDecodeHexString("0014e3178c0f294a9a8f4b304236406507913091df86"), 2, 1.0),
			},
		},
	}

	Eos2BtcOrders = []*common.Order{
		{
			FromAssetID:      &EOS,
			ToAssetID:        &BTC,
			RatioNumerator:   1,
			RatioDenominator: 100,
			Utxo: &common.MovUtxo{
				SourceID:       hashPtr(testutil.MustDecodeHash("27cf8a0877dc858968cc06396fe6aa9e02d15f3e44c862fe29fa5fd50497cf20")),
				SourcePos:      0,
				Amount:         1000,
				ControlProgram: MustCreateP2WMCProgram(BTC, testutil.MustDecodeHexString("00144d0dfc8a0c5ce41d31d4f61d99aff70588bff8bc"), 1, 100.0),
			},
		},
	}

	Btc2EthCancelTxs = []*types.Tx{
		// Btc2EthOrders[0]
		types.NewTx(types.TxData{
			Inputs:  []*types.TxInput{types.NewSpendInput([][]byte{{}, {}, vm.Int64Bytes(2)}, *Btc2EthOrders[0].Utxo.SourceID, *Btc2EthOrders[0].FromAssetID, Btc2EthOrders[0].Utxo.Amount, Btc2EthOrders[0].Utxo.SourcePos, Btc2EthOrders[0].Utxo.ControlProgram)},
			Outputs: []*types.TxOutput{types.NewIntraChainOutput(*Btc2EthOrders[0].FromAssetID, Btc2EthOrders[0].Utxo.Amount, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251"))},
		}),

		// output 2 of MatchedTxs[2]
		types.NewTx(types.TxData{
			Inputs:  []*types.TxInput{types.NewSpendInput([][]byte{{}, {}, vm.Int64Bytes(2)}, *MustNewOrderFromOutput(MatchedTxs[2], 2).Utxo.SourceID, *Eth2BtcOrders[2].FromAssetID, 270, 2, Eth2BtcOrders[2].Utxo.ControlProgram)},
			Outputs: []*types.TxOutput{types.NewIntraChainOutput(*Eth2BtcOrders[2].FromAssetID, Eth2BtcOrders[2].Utxo.Amount, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19255"))},
		}),
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

	Eos2EtcMakerTxs = []*types.Tx{
		// Eos2Etc[0]
		types.NewTx(types.TxData{
			Inputs:  []*types.TxInput{types.NewSpendInput(nil, *Eos2EtcOrders[0].Utxo.SourceID, *Eos2EtcOrders[0].FromAssetID, Eos2EtcOrders[0].Utxo.Amount, Eos2EtcOrders[0].Utxo.SourcePos, []byte{0x51})},
			Outputs: []*types.TxOutput{types.NewIntraChainOutput(*Eos2EtcOrders[0].FromAssetID, Eos2EtcOrders[0].Utxo.Amount, Eos2EtcOrders[0].Utxo.ControlProgram)},
		}),
	}

	Etc2EosMakerTxs = []*types.Tx{
		// Etc2Eos[0]
		types.NewTx(types.TxData{
			Inputs:  []*types.TxInput{types.NewSpendInput(nil, *Etc2EosOrders[0].Utxo.SourceID, *Etc2EosOrders[0].FromAssetID, Etc2EosOrders[0].Utxo.Amount, Etc2EosOrders[0].Utxo.SourcePos, []byte{0x51})},
			Outputs: []*types.TxOutput{types.NewIntraChainOutput(*Etc2EosOrders[0].FromAssetID, Etc2EosOrders[0].Utxo.Amount, Etc2EosOrders[0].Utxo.ControlProgram)},
		}),
	}

	Eth2EosMakerTxs = []*types.Tx{
		// Eth2Eos[0]
		types.NewTx(types.TxData{
			Inputs:  []*types.TxInput{types.NewSpendInput(nil, *Eth2EosOrders[0].Utxo.SourceID, *Eth2EosOrders[0].FromAssetID, Eth2EosOrders[0].Utxo.Amount, Eth2EosOrders[0].Utxo.SourcePos, []byte{0x51})},
			Outputs: []*types.TxOutput{types.NewIntraChainOutput(*Eth2EosOrders[0].FromAssetID, Eth2EosOrders[0].Utxo.Amount, Eth2EosOrders[0].Utxo.ControlProgram)},
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
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 415, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
				// re-order
				types.NewIntraChainOutput(*Btc2EthOrders[0].FromAssetID, 2, Btc2EthOrders[0].Utxo.ControlProgram),
				types.NewIntraChainOutput(*Eth2BtcOrders[1].ToAssetID, 7, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19254")),
				// fee
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 1, RewardProgram),
				types.NewIntraChainOutput(*Eth2BtcOrders[1].ToAssetID, 1, RewardProgram),
			},
		}),

		// full matched transaction from Btc2EthOrders[0], Eth2BtcOrders[0]
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *Btc2EthOrders[0].Utxo.SourceID, *Btc2EthOrders[0].FromAssetID, Btc2EthOrders[0].Utxo.Amount, Btc2EthOrders[0].Utxo.SourcePos, Btc2EthOrders[0].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(1), vm.Int64Bytes(1)}, *Eth2BtcOrders[0].Utxo.SourceID, *Eth2BtcOrders[0].FromAssetID, Eth2BtcOrders[0].Utxo.Amount, Eth2BtcOrders[0].Utxo.SourcePos, Eth2BtcOrders[0].Utxo.ControlProgram),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 499, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
				types.NewIntraChainOutput(*Eth2BtcOrders[0].ToAssetID, 9, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19253")),
				// fee
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 11, RewardProgram),
				types.NewIntraChainOutput(*Eth2BtcOrders[0].ToAssetID, 1, RewardProgram),
			},
		}),

		// partial matched transaction from Btc2EthOrders[0], Eth2BtcOrders[2]
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *Btc2EthOrders[0].Utxo.SourceID, *Btc2EthOrders[0].FromAssetID, Btc2EthOrders[0].Utxo.Amount, Btc2EthOrders[0].Utxo.SourcePos, Btc2EthOrders[0].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(10), vm.Int64Bytes(1), vm.Int64Bytes(0)}, *Eth2BtcOrders[2].Utxo.SourceID, *Eth2BtcOrders[2].FromAssetID, Eth2BtcOrders[2].Utxo.Amount, Eth2BtcOrders[2].Utxo.SourcePos, Eth2BtcOrders[2].Utxo.ControlProgram),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 499, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
				types.NewIntraChainOutput(*Eth2BtcOrders[2].ToAssetID, 9, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19255")),
				// re-order
				types.NewIntraChainOutput(*Eth2BtcOrders[2].FromAssetID, 270, Eth2BtcOrders[2].Utxo.ControlProgram),
				// fee
				types.NewIntraChainOutput(*Eth2BtcOrders[2].FromAssetID, 25, RewardProgram),
				types.NewIntraChainOutput(*Btc2EthOrders[0].FromAssetID, 1, RewardProgram),
				// refund
				types.NewIntraChainOutput(*Eth2BtcOrders[2].FromAssetID, 8, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
				types.NewIntraChainOutput(*Eth2BtcOrders[2].FromAssetID, 8, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19255")),
			},
		}),
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(0)}, *Btc2EthOrders[1].Utxo.SourceID, *Btc2EthOrders[1].FromAssetID, Btc2EthOrders[1].Utxo.Amount, Btc2EthOrders[1].Utxo.SourcePos, Btc2EthOrders[1].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(2), vm.Int64Bytes(1)}, testutil.MustDecodeHash("39bdb7058a0c31fb740af8e3c382bf608efff1b041cd4dd461332722ad24552a"), *Eth2BtcOrders[2].FromAssetID, 270, 2, Eth2BtcOrders[2].Utxo.ControlProgram),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(*Btc2EthOrders[1].ToAssetID, 269, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19252")),
				// re-order
				types.NewIntraChainOutput(*Btc2EthOrders[1].FromAssetID, 15, Btc2EthOrders[1].Utxo.ControlProgram),
				types.NewIntraChainOutput(*Eth2BtcOrders[2].ToAssetID, 4, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19255")),
				// fee
				types.NewIntraChainOutput(*Btc2EthOrders[1].ToAssetID, 1, RewardProgram),
				types.NewIntraChainOutput(*Eth2BtcOrders[2].ToAssetID, 1, RewardProgram),
			},
		}),

		// partial matched transaction from Btc2EthMakerTxs[0], Eth2BtcMakerTxs[1]
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *MustNewOrderFromOutput(Btc2EthMakerTxs[0], 0).Utxo.SourceID, *Btc2EthOrders[0].FromAssetID, Btc2EthOrders[0].Utxo.Amount, 0, Btc2EthOrders[0].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(1), vm.Int64Bytes(1)}, *MustNewOrderFromOutput(Eth2BtcMakerTxs[1], 0).Utxo.SourceID, *Eth2BtcOrders[1].FromAssetID, Eth2BtcOrders[1].Utxo.Amount, 0, Eth2BtcOrders[1].Utxo.ControlProgram),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 415, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
				// re-order
				types.NewIntraChainOutput(*Btc2EthOrders[0].FromAssetID, 2, Btc2EthOrders[0].Utxo.ControlProgram),
				types.NewIntraChainOutput(*Eth2BtcOrders[1].ToAssetID, 7, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19254")),
				// fee
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 1, RewardProgram),
				types.NewIntraChainOutput(*Eth2BtcOrders[1].ToAssetID, 1, RewardProgram),
			},
		}),

		// full matched transaction from Eos2EtcMakerTxs[0] Etc2EosMakerTxs[0]
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *MustNewOrderFromOutput(Eos2EtcMakerTxs[0], 0).Utxo.SourceID, *Eos2EtcOrders[0].FromAssetID, Eos2EtcOrders[0].Utxo.Amount, Eos2EtcOrders[0].Utxo.SourcePos, Eos2EtcOrders[0].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(1), vm.Int64Bytes(1)}, *MustNewOrderFromOutput(Etc2EosMakerTxs[0], 0).Utxo.SourceID, *Etc2EosOrders[0].FromAssetID, Etc2EosOrders[0].Utxo.Amount, Etc2EosOrders[0].Utxo.SourcePos, Etc2EosOrders[0].Utxo.ControlProgram),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(*Eos2EtcOrders[0].ToAssetID, 49, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19255")),
				types.NewIntraChainOutput(*Etc2EosOrders[0].ToAssetID, 99, testutil.MustDecodeHexString("0014df7a97e53bbe278e4e44810b0a760fb472daa9a3")),
				// fee
				types.NewIntraChainOutput(*Eos2EtcOrders[0].ToAssetID, 1, RewardProgram),
				types.NewIntraChainOutput(*Etc2EosOrders[0].ToAssetID, 1, RewardProgram),
			},
		}),

		// cycle matched from Btc2Eth Eth2Eos Eos2Btc
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *Btc2EthOrders[0].Utxo.SourceID, *Btc2EthOrders[0].FromAssetID, Btc2EthOrders[0].Utxo.Amount, Btc2EthOrders[0].Utxo.SourcePos, Btc2EthOrders[0].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(1), vm.Int64Bytes(1)}, *Eth2EosOrders[0].Utxo.SourceID, *Eth2EosOrders[0].FromAssetID, Eth2EosOrders[0].Utxo.Amount, Eth2EosOrders[0].Utxo.SourcePos, Eth2EosOrders[0].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(2), vm.Int64Bytes(1)}, *Eos2BtcOrders[0].Utxo.SourceID, *Eos2BtcOrders[0].FromAssetID, Eos2BtcOrders[0].Utxo.Amount, Eos2BtcOrders[0].Utxo.SourcePos, Eos2BtcOrders[0].Utxo.ControlProgram),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(ETH, 499, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
				types.NewIntraChainOutput(EOS, 999, testutil.MustDecodeHexString("0014e3178c0f294a9a8f4b304236406507913091df86")),
				types.NewIntraChainOutput(BTC, 9, testutil.MustDecodeHexString("00144d0dfc8a0c5ce41d31d4f61d99aff70588bff8bc")),
				// fee
				types.NewIntraChainOutput(ETH, 1, RewardProgram),
				types.NewIntraChainOutput(EOS, 1, RewardProgram),
				types.NewIntraChainOutput(BTC, 1, RewardProgram),
			},
		}),

		// partial matched transaction from MatchedTxs[4], Eth2BtcMakerTxs[0]
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, testutil.MustDecodeHash("ed810e1672c3b9de27a1db23e017e6b9cc23334b6e3dbd25dfe8857e289b7f06"), *Btc2EthOrders[0].FromAssetID, 2, 1, Btc2EthOrders[0].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(1), vm.Int64Bytes(1)}, *MustNewOrderFromOutput(Eth2BtcMakerTxs[0], 0).Utxo.SourceID, *Eth2BtcOrders[0].FromAssetID, Eth2BtcOrders[0].Utxo.Amount, 0, Eth2BtcOrders[0].Utxo.ControlProgram),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 99, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
				types.NewIntraChainOutput(*Eth2BtcOrders[0].ToAssetID, 1, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19253")),
				// re-order
				types.NewIntraChainOutput(*Eth2BtcOrders[0].FromAssetID, 404, Eth2BtcOrders[0].Utxo.ControlProgram),
				// fee
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 1, RewardProgram),
				types.NewIntraChainOutput(*Eth2BtcOrders[0].ToAssetID, 1, RewardProgram),
			},
		}),

		// partial matched transaction from Btc2EthOrders[3], Eth2BtcOrders[2]
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{vm.Int64Bytes(810), vm.Int64Bytes(0), vm.Int64Bytes(0)}, *Btc2EthOrders[3].Utxo.SourceID, *Btc2EthOrders[3].FromAssetID, Btc2EthOrders[3].Utxo.Amount, Btc2EthOrders[3].Utxo.SourcePos, Btc2EthOrders[3].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(2), vm.Int64Bytes(1)}, *Eth2BtcOrders[2].Utxo.SourceID, *Eth2BtcOrders[2].FromAssetID, Eth2BtcOrders[2].Utxo.Amount, Eth2BtcOrders[2].Utxo.SourcePos, Eth2BtcOrders[2].Utxo.ControlProgram),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(*Btc2EthOrders[3].ToAssetID, 809, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19252")),
				// re-order
				types.NewIntraChainOutput(*Btc2EthOrders[3].FromAssetID, 1, Btc2EthOrders[3].Utxo.ControlProgram),
				types.NewIntraChainOutput(*Eth2BtcOrders[2].ToAssetID, 14, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19255")),
				// fee
				types.NewIntraChainOutput(*Btc2EthOrders[3].FromAssetID, 2, RewardProgram),
				types.NewIntraChainOutput(*Eth2BtcOrders[2].ToAssetID, 1, RewardProgram),
			},
		}),

		// full matched transaction from Eos2EtcOrders[0] Etc2EosOrders[0]
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *Eos2EtcOrders[0].Utxo.SourceID, *Eos2EtcOrders[0].FromAssetID, Eos2EtcOrders[0].Utxo.Amount, Eos2EtcOrders[0].Utxo.SourcePos, Eos2EtcOrders[0].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(1), vm.Int64Bytes(1)}, *Etc2EosOrders[0].Utxo.SourceID, *Etc2EosOrders[0].FromAssetID, Etc2EosOrders[0].Utxo.Amount, Etc2EosOrders[0].Utxo.SourcePos, Etc2EosOrders[0].Utxo.ControlProgram),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(*Eos2EtcOrders[0].ToAssetID, 49, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19255")),
				types.NewIntraChainOutput(*Etc2EosOrders[0].ToAssetID, 99, testutil.MustDecodeHexString("0014df7a97e53bbe278e4e44810b0a760fb472daa9a3")),
				// fee
				types.NewIntraChainOutput(*Eos2EtcOrders[0].ToAssetID, 1, RewardProgram),
				types.NewIntraChainOutput(*Etc2EosOrders[0].ToAssetID, 1, RewardProgram),
			},
		}),

		// full matched transaction from Btc2EthOrders[0], Eth2BtcMakerTxs[0]
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *Btc2EthOrders[0].Utxo.SourceID, *Btc2EthOrders[0].FromAssetID, Btc2EthOrders[0].Utxo.Amount, Btc2EthOrders[0].Utxo.SourcePos, Btc2EthOrders[0].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(1), vm.Int64Bytes(1)}, *MustNewOrderFromOutput(Eth2BtcMakerTxs[0], 0).Utxo.SourceID, *Eth2BtcOrders[0].FromAssetID, Eth2BtcOrders[0].Utxo.Amount, 0, Eth2BtcOrders[0].Utxo.ControlProgram),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 499, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
				types.NewIntraChainOutput(*Eth2BtcOrders[0].ToAssetID, 9, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19253")),
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 10, RewardProgram),
				// fee
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 1, RewardProgram),
				types.NewIntraChainOutput(*Eth2BtcOrders[0].ToAssetID, 1, RewardProgram),
			},
		}),

		// full matched transaction from Btc2EthOrders[0] Eth2BtcOrders[3]
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *Btc2EthOrders[0].Utxo.SourceID, *Btc2EthOrders[0].FromAssetID, Btc2EthOrders[0].Utxo.Amount, Btc2EthOrders[0].Utxo.SourcePos, Btc2EthOrders[0].Utxo.ControlProgram),
				types.NewSpendInput([][]byte{vm.Int64Bytes(1), vm.Int64Bytes(1)}, *Eth2BtcOrders[3].Utxo.SourceID, *Eth2BtcOrders[3].FromAssetID, Eth2BtcOrders[3].Utxo.Amount, Eth2BtcOrders[3].Utxo.SourcePos, Eth2BtcOrders[3].Utxo.ControlProgram),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 499, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
				types.NewIntraChainOutput(*Eth2BtcOrders[3].ToAssetID, 3, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19256")),
				// fee
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 25, RewardProgram),
				types.NewIntraChainOutput(*Eth2BtcOrders[3].ToAssetID, 1, RewardProgram),
				// refund
				types.NewIntraChainOutput(*Eth2BtcOrders[3].ToAssetID, 3, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
				types.NewIntraChainOutput(*Eth2BtcOrders[3].ToAssetID, 3, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19256")),
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 38, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
				types.NewIntraChainOutput(*Btc2EthOrders[0].ToAssetID, 38, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19256")),
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
