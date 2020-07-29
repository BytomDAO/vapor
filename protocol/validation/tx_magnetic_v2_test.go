package validation

import (
	"testing"

	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/vm"
	"github.com/bytom/vapor/protocol/vm/vmutil"
	"github.com/bytom/vapor/testutil"
)

func TestMagneticContractV2Tx(t *testing.T) {
	buyerArgs := vmutil.MagneticContractArgs{
		RequestedAsset:   bc.AssetID{V0: 1},
		RatioNumerator:   1,
		RatioDenominator: 2,
		SellerProgram:    testutil.MustDecodeHexString("0014b0d4971493f2b8f7ff02ff8cdbf3605c43aa878f"),
		SellerKey:        testutil.MustDecodeHexString("3b6a27bcceb6a42d62a3a8d02a6f0d73653215771de243a63ac048a18b59da29"),
	}

	sellerArgs := vmutil.MagneticContractArgs{
		RequestedAsset:   bc.AssetID{V0: 2},
		RatioNumerator:   2,
		RatioDenominator: 1,
		SellerProgram:    testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19204"),
		SellerKey:        testutil.MustDecodeHexString("3b6a27bcceb6a42d62a3a8d02a6f0d73653215771de243a63ac048a18b59da29"),
	}

	programBuyer, err := vmutil.P2WMCProgramV2(buyerArgs)
	if err != nil {
		t.Fatal(err)
	}

	programSeller, err := vmutil.P2WMCProgramV2(sellerArgs)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		desc  string
		block *bc.Block
		err   error
	}{
		{
			desc: "contracts all full trade",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Version: 0},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						SerializedSize: 1,
						Inputs: []*types.TxInput{
							types.NewSpendInput([][]byte{vm.Int64Bytes(100), vm.Int64Bytes(0), vm.Int64Bytes(1)}, bc.Hash{V0: 10}, buyerArgs.RequestedAsset, 100000000, 1, programSeller),
							types.NewSpendInput([][]byte{vm.Int64Bytes(100), vm.Int64Bytes(1), vm.Int64Bytes(1)}, bc.Hash{V0: 20}, sellerArgs.RequestedAsset, 200000000, 0, programBuyer),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(sellerArgs.RequestedAsset, 198000000, sellerArgs.SellerProgram),
							types.NewIntraChainOutput(buyerArgs.RequestedAsset, 99000000, buyerArgs.SellerProgram),
							types.NewIntraChainOutput(sellerArgs.RequestedAsset, 2000000, []byte{0x51}),
							types.NewIntraChainOutput(buyerArgs.RequestedAsset, 1000000, []byte{0x51}),
						},
					}),
				},
			},
			err: nil,
		},
		{
			desc: "first contract partial trade, second contract full trade",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Version: 0},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						SerializedSize: 1,
						Inputs: []*types.TxInput{
							types.NewSpendInput([][]byte{vm.Int64Bytes(100000000), vm.Int64Bytes(10), vm.Int64Bytes(0), vm.Int64Bytes(0)}, bc.Hash{V0: 10}, buyerArgs.RequestedAsset, 200000000, 1, programSeller),
							types.NewSpendInput([][]byte{vm.Int64Bytes(10), vm.Int64Bytes(2), vm.Int64Bytes(1)}, bc.Hash{V0: 20}, sellerArgs.RequestedAsset, 100000000, 0, programBuyer),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(sellerArgs.RequestedAsset, 99900000, sellerArgs.SellerProgram),
							types.NewIntraChainOutput(buyerArgs.RequestedAsset, 150000000, programSeller),
							types.NewIntraChainOutput(buyerArgs.RequestedAsset, 49950000, buyerArgs.SellerProgram),
							types.NewIntraChainOutput(sellerArgs.RequestedAsset, 100000, []byte{0x51}),
							types.NewIntraChainOutput(buyerArgs.RequestedAsset, 50000, []byte{0x51}),
						},
					}),
				},
			},
			err: nil,
		},
		{
			desc: "first contract full trade, second contract partial trade",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Version: 0},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						SerializedSize: 1,
						Inputs: []*types.TxInput{
							types.NewSpendInput([][]byte{vm.Int64Bytes(10), vm.Int64Bytes(0), vm.Int64Bytes(1)}, bc.Hash{V0: 10}, buyerArgs.RequestedAsset, 100000000, 1, programSeller),
							types.NewSpendInput([][]byte{vm.Int64Bytes(100000000), vm.Int64Bytes(10), vm.Int64Bytes(1), vm.Int64Bytes(0)}, bc.Hash{V0: 20}, sellerArgs.RequestedAsset, 300000000, 0, programBuyer),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(sellerArgs.RequestedAsset, 199800000, sellerArgs.SellerProgram),
							types.NewIntraChainOutput(buyerArgs.RequestedAsset, 99900000, buyerArgs.SellerProgram),
							types.NewIntraChainOutput(sellerArgs.RequestedAsset, 100000000, programBuyer),
							types.NewIntraChainOutput(sellerArgs.RequestedAsset, 200000, []byte{0x51}),
							types.NewIntraChainOutput(buyerArgs.RequestedAsset, 100000, []byte{0x51}),
						},
					}),
				},
			},
			err: nil,
		},
		{
			desc: "cancel magnetic contract",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Version: 0},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						SerializedSize: 1,
						Inputs: []*types.TxInput{
							types.NewSpendInput([][]byte{testutil.MustDecodeHexString("89bbc39dceb361af5f43e14d4033a81c3a0402d74ecf129325acb33da07bcc70b43cc767ad2da2d32bc00d535d499d2f9b0c6a67ad135f9f5f79b22ef2780402"), vm.Int64Bytes(0), vm.Int64Bytes(2)}, bc.Hash{V0: 10}, buyerArgs.RequestedAsset, 100000000, 0, programSeller),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(buyerArgs.RequestedAsset, 100000000, sellerArgs.SellerProgram),
						},
					}),
				},
			},
			err: nil,
		},
		{
			desc: "wrong signature with cancel magnetic contract",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Version: 0},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						SerializedSize: 1,
						Inputs: []*types.TxInput{
							types.NewSpendInput([][]byte{testutil.MustDecodeHexString("686b983a8de1893ef723144389fd1f07b12b048f52f389faa863243195931d5732dbfd15470b43ed63d5067900718cf94f137073f4a972d277bbd967b022545d"), vm.Int64Bytes(0), vm.Int64Bytes(2)}, bc.Hash{V0: 10}, buyerArgs.RequestedAsset, 100000000, 0, programSeller),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(buyerArgs.RequestedAsset, 100000000, sellerArgs.SellerProgram),
						},
					}),
				},
			},
			err: vm.ErrFalseVMResult,
		},
		{
			desc: "wrong output amount with contracts all full trade",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Version: 0},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						SerializedSize: 1,
						Inputs: []*types.TxInput{
							types.NewSpendInput([][]byte{vm.Int64Bytes(10), vm.Int64Bytes(0), vm.Int64Bytes(1)}, bc.Hash{V0: 10}, buyerArgs.RequestedAsset, 100000000, 1, programSeller),
							types.NewSpendInput([][]byte{vm.Int64Bytes(10), vm.Int64Bytes(1), vm.Int64Bytes(1)}, bc.Hash{V0: 20}, sellerArgs.RequestedAsset, 200000000, 0, programBuyer),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(sellerArgs.RequestedAsset, 200000000, sellerArgs.SellerProgram),
							types.NewIntraChainOutput(buyerArgs.RequestedAsset, 50000000, buyerArgs.SellerProgram),
							types.NewIntraChainOutput(buyerArgs.RequestedAsset, 50000000, []byte{0x55}),
						},
					}),
				},
			},
			err: vm.ErrFalseVMResult,
		},
		{
			desc: "wrong output assetID with contracts all full trade",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Version: 0},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						SerializedSize: 1,
						Inputs: []*types.TxInput{
							types.NewSpendInput([][]byte{vm.Int64Bytes(10), vm.Int64Bytes(0), vm.Int64Bytes(1)}, bc.Hash{V0: 10}, buyerArgs.RequestedAsset, 100000000, 1, programSeller),
							types.NewSpendInput([][]byte{vm.Int64Bytes(10), vm.Int64Bytes(1), vm.Int64Bytes(1)}, bc.Hash{V0: 20}, sellerArgs.RequestedAsset, 200000000, 0, programBuyer),
							types.NewSpendInput(nil, bc.Hash{V0: 30}, bc.AssetID{V0: 1}, 200000000, 0, []byte{0x51}),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(bc.AssetID{V0: 1}, 200000000, sellerArgs.SellerProgram),
							types.NewIntraChainOutput(buyerArgs.RequestedAsset, 100000000, buyerArgs.SellerProgram),
							types.NewIntraChainOutput(sellerArgs.RequestedAsset, 200000000, []byte{0x55}),
						},
					}),
				},
			},
			err: vm.ErrFalseVMResult,
		},
		{
			desc: "wrong output change program with first contract partial trade and second contract full trade",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Version: 0},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						SerializedSize: 1,
						Inputs: []*types.TxInput{
							types.NewSpendInput([][]byte{vm.Int64Bytes(100000000), vm.Int64Bytes(10), vm.Int64Bytes(0), vm.Int64Bytes(0)}, bc.Hash{V0: 10}, buyerArgs.RequestedAsset, 200000000, 1, programSeller),
							types.NewSpendInput([][]byte{vm.Int64Bytes(10), vm.Int64Bytes(2), vm.Int64Bytes(1)}, bc.Hash{V0: 20}, sellerArgs.RequestedAsset, 100000000, 0, programBuyer),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(sellerArgs.RequestedAsset, 99900000, sellerArgs.SellerProgram),
							types.NewIntraChainOutput(buyerArgs.RequestedAsset, 150000000, []byte{0x55}),
							types.NewIntraChainOutput(buyerArgs.RequestedAsset, 49950000, buyerArgs.SellerProgram),
							types.NewIntraChainOutput(sellerArgs.RequestedAsset, 100000, []byte{0x51}),
							types.NewIntraChainOutput(buyerArgs.RequestedAsset, 50000, []byte{0x51}),
						},
					}),
				},
			},
			err: vm.ErrFalseVMResult,
		},
	}

	for i, c := range cases {
		if _, err := ValidateTx(c.block.Transactions[0], c.block); rootErr(err) != c.err {
			t.Errorf("TestMagneticContractV2Tx case #%d (%s) got error %t, want %t", i, c.desc, rootErr(err), c.err)
		}
	}
}

func TestRingMagneticContractV2Tx(t *testing.T) {
	aliceArgs := vmutil.MagneticContractArgs{
		RequestedAsset:   bc.AssetID{V0: 1},
		RatioNumerator:   2,
		RatioDenominator: 1,
		SellerProgram:    testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19204"),
		SellerKey:        testutil.MustDecodeHexString("960ecabafb88ba460a40912841afecebf0e84884178611ac97210e327c0d1173"),
	}

	bobArgs := vmutil.MagneticContractArgs{
		RequestedAsset:   bc.AssetID{V0: 2},
		RatioNumerator:   2,
		RatioDenominator: 1,
		SellerProgram:    testutil.MustDecodeHexString("0014b0d4971493f2b8f7ff02ff8cdbf3605c43aa878f"),
		SellerKey:        testutil.MustDecodeHexString("ad79ec6bd3a6d6dbe4d0ee902afc99a12b9702fb63edce5f651db3081d868b75"),
	}

	jackArgs := vmutil.MagneticContractArgs{
		RequestedAsset:   bc.AssetID{V0: 3},
		RatioNumerator:   1,
		RatioDenominator: 4,
		SellerProgram:    testutil.MustDecodeHexString("0014220f6913bd05821fa80e188d0ba7d633cb77e9fa"),
		SellerKey:        testutil.MustDecodeHexString("9c19a91988c62046c2767bd7e9999b0c142891b9ebf467bfa59210b435cb0de7"),
	}

	aliceProgram, err := vmutil.P2WMCProgramV2(aliceArgs)
	if err != nil {
		t.Fatal(err)
	}

	bobProgram, err := vmutil.P2WMCProgramV2(bobArgs)
	if err != nil {
		t.Fatal(err)
	}

	jackProgram, err := vmutil.P2WMCProgramV2(jackArgs)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		desc  string
		block *bc.Block
		err   error
	}{
		{
			desc: "contracts all full trade",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Version: 0},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						SerializedSize: 1,
						Inputs: []*types.TxInput{
							types.NewSpendInput([][]byte{vm.Int64Bytes(100), vm.Int64Bytes(0), vm.Int64Bytes(1)}, bc.Hash{V0: 10}, jackArgs.RequestedAsset, 100000000, 0, aliceProgram),
							types.NewSpendInput([][]byte{vm.Int64Bytes(100), vm.Int64Bytes(1), vm.Int64Bytes(1)}, bc.Hash{V0: 20}, aliceArgs.RequestedAsset, 200000000, 0, bobProgram),
							types.NewSpendInput([][]byte{vm.Int64Bytes(100), vm.Int64Bytes(2), vm.Int64Bytes(1)}, bc.Hash{V0: 30}, bobArgs.RequestedAsset, 400000000, 0, jackProgram),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(aliceArgs.RequestedAsset, 198000000, aliceArgs.SellerProgram),
							types.NewIntraChainOutput(bobArgs.RequestedAsset, 396000000, bobArgs.SellerProgram),
							types.NewIntraChainOutput(jackArgs.RequestedAsset, 99000000, jackArgs.SellerProgram),
							types.NewIntraChainOutput(aliceArgs.RequestedAsset, 2000000, []byte{0x51}),
							types.NewIntraChainOutput(bobArgs.RequestedAsset, 4000000, []byte{0x51}),
							types.NewIntraChainOutput(jackArgs.RequestedAsset, 1000000, []byte{0x51}),
						},
					}),
				},
			},
			err: nil,
		},
	}

	for i, c := range cases {
		if _, err := ValidateTx(c.block.Transactions[0], c.block); rootErr(err) != c.err {
			t.Errorf("TestRingMagneticContractV2Tx case #%d (%s) got error %t, want %t", i, c.desc, rootErr(err), c.err)
		}
	}
}
