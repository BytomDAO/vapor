package validation

import (
	"fmt"
	"testing"

	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm"
	"github.com/vapor/protocol/vm/vmutil"
	"github.com/vapor/testutil"
)

func TestCheckOutput(t *testing.T) {
	tx := types.NewTx(types.TxData{
		Inputs: []*types.TxInput{
			types.NewSpendInput(nil, bc.Hash{}, bc.NewAssetID([32]byte{1}), 5, 1, []byte("spendprog")),
		},
		Outputs: []*types.TxOutput{
			types.NewIntraChainOutput(bc.NewAssetID([32]byte{3}), 8, []byte("wrongprog")),
			types.NewIntraChainOutput(bc.NewAssetID([32]byte{3}), 8, []byte("controlprog")),
			types.NewIntraChainOutput(bc.NewAssetID([32]byte{2}), 8, []byte("controlprog")),
			types.NewIntraChainOutput(bc.NewAssetID([32]byte{2}), 7, []byte("controlprog")),
			types.NewIntraChainOutput(bc.NewAssetID([32]byte{2}), 7, []byte("controlprog")),
		},
	})

	txCtx := &entryContext{
		entry:   tx.Tx.Entries[tx.Tx.InputIDs[0]],
		entries: tx.Tx.Entries,
	}

	cases := []struct {
		// args to CheckOutput
		index     uint64
		amount    uint64
		assetID   []byte
		vmVersion uint64
		code      []byte

		wantErr error
		wantOk  bool
	}{
		{
			index:     4,
			amount:    7,
			assetID:   append([]byte{2}, make([]byte, 31)...),
			vmVersion: 1,
			code:      []byte("controlprog"),
			wantOk:    true,
		},
		{
			index:     3,
			amount:    7,
			assetID:   append([]byte{2}, make([]byte, 31)...),
			vmVersion: 1,
			code:      []byte("controlprog"),
			wantOk:    true,
		},
		{
			index:     0,
			amount:    1,
			assetID:   append([]byte{9}, make([]byte, 31)...),
			vmVersion: 1,
			code:      []byte("missingprog"),
			wantOk:    false,
		},
		{
			index:     5,
			amount:    7,
			assetID:   append([]byte{2}, make([]byte, 31)...),
			vmVersion: 1,
			code:      []byte("controlprog"),
			wantErr:   vm.ErrBadValue,
		},
	}

	for i, test := range cases {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			gotOk, err := txCtx.checkOutput(test.index, test.amount, test.assetID, test.vmVersion, test.code, false)
			if g := errors.Root(err); g != test.wantErr {
				t.Errorf("checkOutput(%v, %v, %x, %v, %x) err = %v, want %v", test.index, test.amount, test.assetID, test.vmVersion, test.code, g, test.wantErr)
				return
			}
			if gotOk != test.wantOk {
				t.Errorf("checkOutput(%v, %v, %x, %v, %x) ok = %t, want %v", test.index, test.amount, test.assetID, test.vmVersion, test.code, gotOk, test.wantOk)
			}

		})
	}
}

func TestExecMagneticContractTx(t *testing.T) {
	buyerArgs := vmutil.MagneticContractArgs{
		RequestedAsset:   bc.AssetID{V0: 1},
		RatioMolecule:    1,
		RatioDenominator: 2,
		SellerProgram:    []byte{0x51},
		SellerKey:        testutil.MustDecodeHexString("a7208aa39c7629ee9e585b95dfffc4d61f32e54dee28f23f9ae419d5088ba6e2"),
	}

	sellerArgs := vmutil.MagneticContractArgs{
		RequestedAsset:   bc.AssetID{V0: 2},
		RatioMolecule:    2,
		RatioDenominator: 1,
		SellerProgram:    []byte{0x52},
		SellerKey:        testutil.MustDecodeHexString("74f7f67ae4bb711c62f560a6a8c259f2b7ceeb1e32d57c1f31f32e256874caa9"),
	}

	programBuyer, _ := vmutil.P2WMCProgram(buyerArgs)
	programSeller, _ := vmutil.P2WMCProgram(sellerArgs)
	cases := []struct {
		desc    string
		vs      *validationState
		wantErr error
	}{
		{
			desc: "contracts all full trade",
			vs: &validationState{
				block: &bc.Block{
					BlockHeader: &bc.BlockHeader{
						Height: 1,
					},
				},
				tx: types.MapTx(&types.TxData{
					Inputs: []*types.TxInput{
						types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, bc.Hash{V0: 10}, buyerArgs.RequestedAsset, 100000000, 1, programSeller),
						types.NewSpendInput([][]byte{vm.Int64Bytes(1), vm.Int64Bytes(1)}, bc.Hash{V0: 20}, sellerArgs.RequestedAsset, 200000000, 0, programBuyer),
					},
					Outputs: []*types.TxOutput{
						types.NewIntraChainOutput(sellerArgs.RequestedAsset, 200000000, sellerArgs.SellerProgram),
						types.NewIntraChainOutput(buyerArgs.RequestedAsset, 100000000, buyerArgs.SellerProgram),
					},
				}),
			},
			wantErr: nil,
		},
		{
			desc: "first contract partial trade, second contract full trade",
			vs: &validationState{
				block: &bc.Block{
					BlockHeader: &bc.BlockHeader{
						Height: 1,
					},
				},
				tx: types.MapTx(&types.TxData{
					Inputs: []*types.TxInput{
						types.NewSpendInput([][]byte{vm.Int64Bytes(100000000), vm.Int64Bytes(0), vm.Int64Bytes(0)}, bc.Hash{V0: 10}, buyerArgs.RequestedAsset, 200000000, 1, programSeller),
						types.NewSpendInput([][]byte{vm.Int64Bytes(2), vm.Int64Bytes(1)}, bc.Hash{V0: 20}, sellerArgs.RequestedAsset, 100000000, 0, programBuyer),
					},
					Outputs: []*types.TxOutput{
						types.NewIntraChainOutput(sellerArgs.RequestedAsset, 100000000, sellerArgs.SellerProgram),
						types.NewIntraChainOutput(buyerArgs.RequestedAsset, 150000000, programSeller),
						types.NewIntraChainOutput(buyerArgs.RequestedAsset, 50000000, buyerArgs.SellerProgram),
					},
				}),
			},
			wantErr: nil,
		},
		{
			desc: "first contract full trade, second contract partial trade",
			vs: &validationState{
				block: &bc.Block{
					BlockHeader: &bc.BlockHeader{
						Height: 1,
					},
				},
				tx: types.MapTx(&types.TxData{
					Inputs: []*types.TxInput{
						types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, bc.Hash{V0: 10}, buyerArgs.RequestedAsset, 100000000, 1, programSeller),
						types.NewSpendInput([][]byte{vm.Int64Bytes(100000000), vm.Int64Bytes(1), vm.Int64Bytes(0)}, bc.Hash{V0: 20}, sellerArgs.RequestedAsset, 300000000, 0, programBuyer),
					},
					Outputs: []*types.TxOutput{
						types.NewIntraChainOutput(sellerArgs.RequestedAsset, 200000000, sellerArgs.SellerProgram),
						types.NewIntraChainOutput(buyerArgs.RequestedAsset, 100000000, buyerArgs.SellerProgram),
						types.NewIntraChainOutput(sellerArgs.RequestedAsset, 100000000, programBuyer),
					},
				}),
			},
			wantErr: nil,
		},
	}

	for _, c := range cases {
		for _, entry := range c.vs.tx.Entries {
			if e, ok := entry.(*bc.Spend); ok {
				spentOutput, err := c.vs.tx.IntraChainOutput(*e.SpentOutputId)
				if err != nil {
					t.Fatal(err)
				}

				if _, err := vm.Verify(NewTxVMContext(c.vs, e, spentOutput.ControlProgram, e.WitnessArguments), 100000000); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
