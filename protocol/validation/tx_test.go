package validation

import (
	"math"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/vapor/consensus"
	"github.com/vapor/crypto/sha3pool"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm"
	"github.com/vapor/protocol/vm/vmutil"
	"github.com/vapor/testutil"
)

func init() {
	spew.Config.DisableMethods = true
}

func TestGasStatus(t *testing.T) {
	cases := []struct {
		input  *GasState
		output *GasState
		f      func(*GasState) error
		err    error
	}{
		{
			input: &GasState{
				GasLeft:  10000,
				GasUsed:  0,
				BTMValue: 0,
			},
			output: &GasState{
				GasLeft:  10000 / consensus.VMGasRate,
				GasUsed:  0,
				BTMValue: 10000,
			},
			f: func(input *GasState) error {
				return input.setGas(10000, 0)
			},
			err: nil,
		},
		{
			input: &GasState{
				GasLeft:  10000,
				GasUsed:  0,
				BTMValue: 0,
			},
			output: &GasState{
				GasLeft:  10000,
				GasUsed:  0,
				BTMValue: 0,
			},
			f: func(input *GasState) error {
				return input.setGas(-10000, 0)
			},
			err: ErrGasCalculate,
		},
		{
			input: &GasState{
				GasLeft:  consensus.DefaultGasCredit,
				GasUsed:  0,
				BTMValue: 0,
			},
			output: &GasState{
				GasLeft:  200000,
				GasUsed:  0,
				BTMValue: 80000000000,
			},
			f: func(input *GasState) error {
				return input.setGas(80000000000, 0)
			},
			err: nil,
		},
		{
			input: &GasState{
				GasLeft:  consensus.DefaultGasCredit,
				GasUsed:  0,
				BTMValue: 0,
			},
			output: &GasState{
				GasLeft:  200000,
				GasUsed:  0,
				BTMValue: math.MaxInt64,
			},
			f: func(input *GasState) error {
				return input.setGas(math.MaxInt64, 0)
			},
			err: nil,
		},
		{
			input: &GasState{
				GasLeft:  10000,
				GasUsed:  0,
				BTMValue: 0,
			},
			output: &GasState{
				GasLeft:  10000,
				GasUsed:  0,
				BTMValue: 0,
			},
			f: func(input *GasState) error {
				return input.updateUsage(-1)
			},
			err: ErrGasCalculate,
		},
		{
			input: &GasState{
				GasLeft:  10000,
				GasUsed:  0,
				BTMValue: 0,
			},
			output: &GasState{
				GasLeft:  9999,
				GasUsed:  1,
				BTMValue: 0,
			},
			f: func(input *GasState) error {
				return input.updateUsage(9999)
			},
			err: nil,
		},
		{
			input: &GasState{
				GasLeft:  -10000,
				GasUsed:  0,
				BTMValue: 0,
			},
			output: &GasState{
				GasLeft:  -10000,
				GasUsed:  0,
				BTMValue: 0,
			},
			f: func(input *GasState) error {
				return input.updateUsage(math.MaxInt64)
			},
			err: ErrGasCalculate,
		},
		{
			input: &GasState{
				GasLeft:    1000,
				GasUsed:    10,
				StorageGas: 1000,
				GasValid:   false,
			},
			output: &GasState{
				GasLeft:    0,
				GasUsed:    1010,
				StorageGas: 1000,
				GasValid:   true,
			},
			f: func(input *GasState) error {
				return input.setGasValid()
			},
			err: nil,
		},
		{
			input: &GasState{
				GasLeft:    900,
				GasUsed:    10,
				StorageGas: 1000,
				GasValid:   false,
			},
			output: &GasState{
				GasLeft:    -100,
				GasUsed:    10,
				StorageGas: 1000,
				GasValid:   false,
			},
			f: func(input *GasState) error {
				return input.setGasValid()
			},
			err: ErrGasCalculate,
		},
		{
			input: &GasState{
				GasLeft:    1000,
				GasUsed:    math.MaxInt64,
				StorageGas: 1000,
				GasValid:   false,
			},
			output: &GasState{
				GasLeft:    0,
				GasUsed:    0,
				StorageGas: 1000,
				GasValid:   false,
			},
			f: func(input *GasState) error {
				return input.setGasValid()
			},
			err: ErrGasCalculate,
		},
		{
			input: &GasState{
				GasLeft:    math.MinInt64,
				GasUsed:    0,
				StorageGas: 1000,
				GasValid:   false,
			},
			output: &GasState{
				GasLeft:    0,
				GasUsed:    0,
				StorageGas: 1000,
				GasValid:   false,
			},
			f: func(input *GasState) error {
				return input.setGasValid()
			},
			err: ErrGasCalculate,
		},
	}

	for i, c := range cases {
		err := c.f(c.input)

		if rootErr(err) != c.err {
			t.Errorf("case %d: got error %s, want %s", i, err, c.err)
		} else if *c.input != *c.output {
			t.Errorf("case %d: gasStatus %v, want %v;", i, c.input, c.output)
		}
	}
}

func TestOverflow(t *testing.T) {
	sourceID := &bc.Hash{V0: 9999}
	ctrlProgram := []byte{byte(vm.OP_TRUE)}
	newTx := func(inputs []uint64, outputs []uint64) *bc.Tx {
		txInputs := make([]*types.TxInput, 0, len(inputs))
		txOutputs := make([]*types.TxOutput, 0, len(outputs))

		for _, amount := range inputs {
			txInput := types.NewSpendInput(nil, *sourceID, *consensus.BTMAssetID, amount, 0, ctrlProgram)
			txInputs = append(txInputs, txInput)
		}

		for _, amount := range outputs {
			txOutput := types.NewIntraChainOutput(*consensus.BTMAssetID, amount, ctrlProgram)
			txOutputs = append(txOutputs, txOutput)
		}

		txData := &types.TxData{
			Version:        1,
			SerializedSize: 100,
			TimeRange:      0,
			Inputs:         txInputs,
			Outputs:        txOutputs,
		}
		return types.MapTx(txData)
	}

	cases := []struct {
		inputs  []uint64
		outputs []uint64
		err     error
	}{
		{
			inputs:  []uint64{math.MaxUint64, 1},
			outputs: []uint64{0},
			err:     ErrOverflow,
		},
		{
			inputs:  []uint64{math.MaxUint64, math.MaxUint64},
			outputs: []uint64{0},
			err:     ErrOverflow,
		},
		{
			inputs:  []uint64{math.MaxUint64, math.MaxUint64 - 1},
			outputs: []uint64{0},
			err:     ErrOverflow,
		},
		{
			inputs:  []uint64{math.MaxInt64, 1},
			outputs: []uint64{0},
			err:     ErrOverflow,
		},
		{
			inputs:  []uint64{math.MaxInt64, math.MaxInt64},
			outputs: []uint64{0},
			err:     ErrOverflow,
		},
		{
			inputs:  []uint64{math.MaxInt64, math.MaxInt64 - 1},
			outputs: []uint64{0},
			err:     ErrOverflow,
		},
		{
			inputs:  []uint64{0},
			outputs: []uint64{math.MaxUint64},
			err:     ErrOverflow,
		},
		{
			inputs:  []uint64{0},
			outputs: []uint64{math.MaxInt64},
			err:     ErrGasCalculate,
		},
		{
			inputs:  []uint64{math.MaxInt64 - 1},
			outputs: []uint64{math.MaxInt64},
			err:     ErrGasCalculate,
		},
	}

	for i, c := range cases {
		tx := newTx(c.inputs, c.outputs)
		if _, err := ValidateTx(tx, mockBlock()); rootErr(err) != c.err {
			t.Fatalf("case %d test failed, want %s, have %s", i, c.err, rootErr(err))
		}
	}
}

func TestTxValidation(t *testing.T) {
	var (
		tx      *bc.Tx
		vs      *validationState
		fixture *txFixture

		// the mux from tx, pulled out for convenience
		mux *bc.Mux
	)

	addCoinbase := func(assetID *bc.AssetID, amount uint64, arbitrary []byte) {
		coinbase := bc.NewCoinbase(arbitrary)
		txOutput := types.NewIntraChainOutput(*assetID, amount, []byte{byte(vm.OP_TRUE)})
		assetAmount := txOutput.AssetAmount()
		muxID := getMuxID(tx)
		coinbase.SetDestination(muxID, &assetAmount, uint64(len(mux.Sources)))
		coinbaseID := bc.EntryID(coinbase)
		tx.Entries[coinbaseID] = coinbase

		mux.Sources = append(mux.Sources, &bc.ValueSource{
			Ref:   &coinbaseID,
			Value: &assetAmount,
		})

		src := &bc.ValueSource{
			Ref:      muxID,
			Value:    &assetAmount,
			Position: uint64(len(tx.ResultIds)),
		}
		prog := &bc.Program{txOutput.VMVersion(), txOutput.ControlProgram()}
		output := bc.NewIntraChainOutput(src, prog, uint64(len(tx.ResultIds)))
		outputID := bc.EntryID(output)
		tx.Entries[outputID] = output

		dest := &bc.ValueDestination{
			Value:    src.Value,
			Ref:      &outputID,
			Position: 0,
		}
		mux.WitnessDestinations = append(mux.WitnessDestinations, dest)
		tx.ResultIds = append(tx.ResultIds, &outputID)
		vs.block.Transactions = append(vs.block.Transactions, vs.tx)
	}

	cases := []struct {
		desc string // description of the test case
		f    func() // function to adjust tx, vs, and/or mux
		err  error  // expected error
	}{
		{
			desc: "base case",
		},
		{
			desc: "unbalanced mux amounts",
			f: func() {
				mux.WitnessDestinations[0].Value.Amount++
			},
			err: ErrUnbalanced,
		},
		{
			desc: "balanced mux amounts",
			f: func() {
				mux.Sources[1].Value.Amount++
				mux.WitnessDestinations[0].Value.Amount++
			},
			err: nil,
		},
		{
			desc: "underflowing mux destination amounts",
			f: func() {
				mux.WitnessDestinations[0].Value.Amount = math.MaxInt64
				out := tx.Entries[*mux.WitnessDestinations[0].Ref].(*bc.IntraChainOutput)
				out.Source.Value.Amount = math.MaxInt64
				mux.WitnessDestinations[1].Value.Amount = math.MaxInt64
				out = tx.Entries[*mux.WitnessDestinations[1].Ref].(*bc.IntraChainOutput)
				out.Source.Value.Amount = math.MaxInt64
			},
			err: ErrOverflow,
		},
		{
			desc: "unbalanced mux assets",
			f: func() {
				mux.Sources[1].Value.AssetId = newAssetID(255)
				sp := tx.Entries[*mux.Sources[1].Ref].(*bc.Spend)
				sp.WitnessDestination.Value.AssetId = newAssetID(255)
			},
			err: ErrUnbalanced,
		},
		{
			desc: "mismatched output source / mux dest position",
			f: func() {
				tx.Entries[*tx.ResultIds[0]].(*bc.IntraChainOutput).Source.Position = 1
			},
			err: ErrMismatchedPosition,
		},
		{
			desc: "mismatched input dest / mux source position",
			f: func() {
				mux.Sources[0].Position = 1
			},
			err: ErrMismatchedPosition,
		},
		{
			desc: "mismatched output source and mux dest",
			f: func() {
				// For this test, it's necessary to construct a mostly
				// identical second transaction in order to get a similar but
				// not equal output entry for the mux to falsely point
				// to. That entry must be added to the first tx's Entries map.
				fixture2 := sample(t, fixture)
				tx2 := types.NewTx(*fixture2.tx).Tx
				out2ID := tx2.ResultIds[0]
				out2 := tx2.Entries[*out2ID].(*bc.IntraChainOutput)
				tx.Entries[*out2ID] = out2
				mux.WitnessDestinations[0].Ref = out2ID
			},
			err: ErrMismatchedReference,
		},
		{
			desc: "invalid mux destination position",
			f: func() {
				mux.WitnessDestinations[0].Position = 1
			},
			err: ErrPosition,
		},
		{
			desc: "mismatched mux dest value / output source value",
			f: func() {
				outID := tx.ResultIds[0]
				out := tx.Entries[*outID].(*bc.IntraChainOutput)
				mux.WitnessDestinations[0].Value = &bc.AssetAmount{
					AssetId: out.Source.Value.AssetId,
					Amount:  out.Source.Value.Amount + 1,
				}
				mux.Sources[0].Value.Amount++ // the mux must still balance
			},
			err: ErrMismatchedValue,
		},
		{
			desc: "empty tx results",
			f: func() {
				tx.ResultIds = nil
			},
			err: ErrEmptyResults,
		},
		{
			desc: "empty tx results, but that's OK",
			f: func() {
				tx.Version = 2
				tx.ResultIds = nil
			},
		},
		{
			desc: "spend control program failure",
			f: func() {
				spend := txSpend(t, tx, 1)
				spend.WitnessArguments[0] = []byte{}
			},
			err: vm.ErrFalseVMResult,
		},
		{
			desc: "mismatched spent source/witness value",
			f: func() {
				spend := txSpend(t, tx, 1)
				spentOutput := tx.Entries[*spend.SpentOutputId].(*bc.IntraChainOutput)
				spentOutput.Source.Value = &bc.AssetAmount{
					AssetId: spend.WitnessDestination.Value.AssetId,
					Amount:  spend.WitnessDestination.Value.Amount + 1,
				}
			},
			err: ErrMismatchedValue,
		},
		{
			desc: "gas out of limit",
			f: func() {
				vs.tx.SerializedSize = 10000000
			},
			err: ErrOverGasCredit,
		},
		{
			desc: "can't find gas spend input in entries",
			f: func() {
				spendID := mux.Sources[len(mux.Sources)-1].Ref
				delete(tx.Entries, *spendID)
				mux.Sources = mux.Sources[:len(mux.Sources)-1]
			},
			err: bc.ErrMissingEntry,
		},
		{
			desc: "no gas spend input",
			f: func() {
				spendID := mux.Sources[len(mux.Sources)-1].Ref
				delete(tx.Entries, *spendID)
				mux.Sources = mux.Sources[:len(mux.Sources)-1]
				tx.GasInputIDs = nil
				vs.gasStatus.GasLeft = 0
			},
			err: vm.ErrRunLimitExceeded,
		},
		{
			desc: "no gas spend input, but set gas left, so it's ok",
			f: func() {
				spendID := mux.Sources[len(mux.Sources)-1].Ref
				delete(tx.Entries, *spendID)
				mux.Sources = mux.Sources[:len(mux.Sources)-1]
				tx.GasInputIDs = nil
			},
			err: nil,
		},
		{
			desc: "mismatched gas spend input destination amount/prevout source amount",
			f: func() {
				spendID := mux.Sources[len(mux.Sources)-1].Ref
				spend := tx.Entries[*spendID].(*bc.Spend)
				spend.WitnessDestination.Value = &bc.AssetAmount{
					AssetId: spend.WitnessDestination.Value.AssetId,
					Amount:  spend.WitnessDestination.Value.Amount + 1,
				}
			},
			err: ErrMismatchedValue,
		},
		{
			desc: "normal coinbase tx",
			f: func() {
				addCoinbase(consensus.BTMAssetID, 100000, nil)
			},
			err: nil,
		},
		{
			desc: "invalid coinbase tx asset id",
			f: func() {
				addCoinbase(&bc.AssetID{V1: 100}, 100000, nil)
			},
			err: ErrWrongCoinbaseAsset,
		},
		{
			desc: "coinbase tx is not first tx in block",
			f: func() {
				addCoinbase(consensus.BTMAssetID, 100000, nil)
				vs.block.Transactions[0] = nil
			},
			err: ErrWrongCoinbaseTransaction,
		},
		{
			desc: "coinbase arbitrary size out of limit",
			f: func() {
				arbitrary := make([]byte, consensus.CoinbaseArbitrarySizeLimit+1)
				addCoinbase(consensus.BTMAssetID, 100000, arbitrary)
			},
			err: ErrCoinbaseArbitraryOversize,
		},
		{
			desc: "normal retirement output",
			f: func() {
				outputID := tx.ResultIds[0]
				output := tx.Entries[*outputID].(*bc.IntraChainOutput)
				retirement := bc.NewRetirement(output.Source, output.Ordinal)
				retirementID := bc.EntryID(retirement)
				tx.Entries[retirementID] = retirement
				delete(tx.Entries, *outputID)
				tx.ResultIds[0] = &retirementID
				mux.WitnessDestinations[0].Ref = &retirementID
			},
			err: nil,
		},
		{
			desc: "ordinal doesn't matter for prevouts",
			f: func() {
				spend := txSpend(t, tx, 1)
				prevout := tx.Entries[*spend.SpentOutputId].(*bc.IntraChainOutput)
				newPrevout := bc.NewIntraChainOutput(prevout.Source, prevout.ControlProgram, 10)
				hash := bc.EntryID(newPrevout)
				spend.SpentOutputId = &hash
			},
			err: nil,
		},
		{
			desc: "mux witness destination have no source",
			f: func() {
				dest := &bc.ValueDestination{
					Value: &bc.AssetAmount{
						AssetId: &bc.AssetID{V2: 1000},
						Amount:  100,
					},
					Ref:      mux.WitnessDestinations[0].Ref,
					Position: 0,
				}
				mux.WitnessDestinations = append(mux.WitnessDestinations, dest)
			},
			err: ErrNoSource,
		},
	}

	for i, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			fixture = sample(t, nil)
			tx = types.NewTx(*fixture.tx).Tx
			vs = &validationState{
				block:   mockBlock(),
				tx:      tx,
				entryID: tx.ID,
				gasStatus: &GasState{
					GasLeft: int64(80000),
					GasUsed: 0,
				},
				cache: make(map[bc.Hash]error),
			}
			muxID := getMuxID(tx)
			mux = tx.Entries[*muxID].(*bc.Mux)

			if c.f != nil {
				c.f()
			}
			err := checkValid(vs, tx.TxHeader)

			if rootErr(err) != c.err {
				t.Errorf("case #%d (%s) got error %s, want %s; validationState is:\n%s", i, c.desc, err, c.err, spew.Sdump(vs))
			}
		})
	}
}

// TestCoinbase test the coinbase transaction is valid (txtest#1016)
func TestCoinbase(t *testing.T) {
	cp, _ := vmutil.DefaultCoinbaseProgram()
	retire, _ := vmutil.RetireProgram([]byte{})
	CbTx := types.MapTx(&types.TxData{
		SerializedSize: 1,
		Inputs: []*types.TxInput{
			types.NewCoinbaseInput(nil),
		},
		Outputs: []*types.TxOutput{
			types.NewIntraChainOutput(*consensus.BTMAssetID, 888, cp),
		},
	})

	cases := []struct {
		block    *bc.Block
		txIndex  int
		GasValid bool
		err      error
	}{
		{
			block: &bc.Block{
				BlockHeader:  &bc.BlockHeader{Height: 666},
				Transactions: []*bc.Tx{CbTx},
			},
			txIndex:  0,
			GasValid: true,
			err:      nil,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Height: 666},
				Transactions: []*bc.Tx{
					CbTx,
					types.MapTx(&types.TxData{
						SerializedSize: 1,
						Inputs: []*types.TxInput{
							types.NewCoinbaseInput(nil),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(*consensus.BTMAssetID, 888, cp),
						},
					}),
				},
			},
			txIndex:  1,
			GasValid: false,
			err:      ErrWrongCoinbaseTransaction,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Height: 666},
				Transactions: []*bc.Tx{
					CbTx,
					types.MapTx(&types.TxData{
						SerializedSize: 1,
						Inputs: []*types.TxInput{
							types.NewCoinbaseInput(nil),
							types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(*consensus.BTMAssetID, 888, cp),
							types.NewIntraChainOutput(*consensus.BTMAssetID, 90000000, cp),
						},
					}),
				},
			},
			txIndex:  1,
			GasValid: false,
			err:      ErrWrongCoinbaseTransaction,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Height: 666},
				Transactions: []*bc.Tx{
					CbTx,
					types.MapTx(&types.TxData{
						SerializedSize: 1,
						Inputs: []*types.TxInput{
							types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp),
							types.NewCoinbaseInput(nil),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(*consensus.BTMAssetID, 888, cp),
							types.NewIntraChainOutput(*consensus.BTMAssetID, 90000000, cp),
						},
					}),
				},
			},
			txIndex:  1,
			GasValid: false,
			err:      ErrWrongCoinbaseTransaction,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Height: 666},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						SerializedSize: 1,
						Inputs: []*types.TxInput{
							types.NewCoinbaseInput(nil),
							types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(*consensus.BTMAssetID, 888, cp),
							types.NewIntraChainOutput(*consensus.BTMAssetID, 90000000, cp),
						},
					}),
				},
			},
			txIndex:  0,
			GasValid: true,
			err:      nil,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Height: 666},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						SerializedSize: 1,
						Inputs: []*types.TxInput{
							types.NewCoinbaseInput(nil),
							types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, retire),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(*consensus.BTMAssetID, 888, cp),
							types.NewIntraChainOutput(*consensus.BTMAssetID, 90000000, cp),
						},
					}),
				},
			},
			txIndex:  0,
			GasValid: false,
			err:      vm.ErrReturn,
		},
	}

	for i, c := range cases {
		gasStatus, err := ValidateTx(c.block.Transactions[c.txIndex], c.block)

		if rootErr(err) != c.err {
			t.Errorf("#%d got error %s, want %s", i, err, c.err)
		}
		if c.GasValid != gasStatus.GasValid {
			t.Errorf("#%d got GasValid %t, want %t", i, gasStatus.GasValid, c.GasValid)
		}
	}
}

// TestTimeRange test the checkTimeRange function (txtest#1004)
func TestTimeRange(t *testing.T) {
	cases := []struct {
		timeRange uint64
		err       bool
	}{
		{
			timeRange: 0,
			err:       false,
		},
		{
			timeRange: 334,
			err:       false,
		},
		{
			timeRange: 332,
			err:       true,
		},
		{
			timeRange: 1521625824,
			err:       false,
		},
	}

	block := &bc.Block{
		BlockHeader: &bc.BlockHeader{
			Height:    333,
			Timestamp: 1521625823000,
		},
	}

	tx := types.MapTx(&types.TxData{
		SerializedSize: 1,
		TimeRange:      0,
		Inputs: []*types.TxInput{
			mockGasTxInput(),
		},
		Outputs: []*types.TxOutput{
			types.NewIntraChainOutput(*consensus.BTMAssetID, 1, []byte{0x6a}),
		},
	})

	for i, c := range cases {
		tx.TimeRange = c.timeRange
		if _, err := ValidateTx(tx, block); (err != nil) != c.err {
			t.Errorf("#%d got error %t, want %t", i, !c.err, c.err)
		}
	}
}

func TestValidateTxVersion(t *testing.T) {
	cases := []struct {
		desc  string
		block *bc.Block
		err   error
	}{
		{
			desc: "tx version greater than 1 (txtest#1001)",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Version: 1},
				Transactions: []*bc.Tx{
					{TxHeader: &bc.TxHeader{Version: 2}},
				},
			},
			err: ErrTxVersion,
		},
		{
			desc: "tx version equals 0 (txtest#1002)",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Version: 1},
				Transactions: []*bc.Tx{
					{TxHeader: &bc.TxHeader{Version: 0}},
				},
			},
			err: ErrTxVersion,
		},
		{
			desc: "tx version equals max uint64 (txtest#1003)",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{Version: 1},
				Transactions: []*bc.Tx{
					{TxHeader: &bc.TxHeader{Version: math.MaxUint64}},
				},
			},
			err: ErrTxVersion,
		},
	}

	for i, c := range cases {
		if _, err := ValidateTx(c.block.Transactions[0], c.block); rootErr(err) != c.err {
			t.Errorf("case #%d (%s) got error %t, want %t", i, c.desc, err, c.err)
		}
	}
}

// A txFixture is returned by sample (below) to produce a sample
// transaction, which takes a separate, optional _input_ txFixture to
// affect the transaction that's built. The components of the
// transaction are the fields of txFixture.
type txFixture struct {
	initialBlockID bc.Hash
	issuanceProg   bc.Program
	issuanceArgs   [][]byte
	assetDef       []byte
	assetID        bc.AssetID
	txVersion      uint64
	txInputs       []*types.TxInput
	txOutputs      []*types.TxOutput
	tx             *types.TxData
}

// Produces a sample transaction in a txFixture object (see above). A
// separate input txFixture can be used to alter the transaction
// that's created.
//
// The output of this function can be used as the input to a
// subsequent call to make iterative refinements to a test object.
//
// The default transaction produced is valid and has three inputs:
//  - an issuance of 10 units
//  - a spend of 20 units
//  - a spend of 40 units
// and two outputs, one of 25 units and one of 45 units.
// All amounts are denominated in the same asset.
//
// The issuance program for the asset requires two numbers as
// arguments that add up to 5. The prevout control programs require
// two numbers each, adding to 9 and 13, respectively.
//
// The min and max times for the transaction are now +/- one minute.
func sample(tb testing.TB, in *txFixture) *txFixture {
	var result txFixture
	if in != nil {
		result = *in
	}

	if result.initialBlockID.IsZero() {
		result.initialBlockID = *newHash(1)
	}
	if testutil.DeepEqual(result.issuanceProg, bc.Program{}) {
		prog, err := vm.Assemble("ADD 5 NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		result.issuanceProg = bc.Program{VmVersion: 1, Code: prog}
	}
	if len(result.issuanceArgs) == 0 {
		result.issuanceArgs = [][]byte{{2}, {3}}
	}
	if len(result.assetDef) == 0 {
		result.assetDef = []byte{2}
	}
	if result.assetID.IsZero() {
		result.assetID = bc.AssetID{V0: 9999}
	}

	if result.txVersion == 0 {
		result.txVersion = 1
	}
	if len(result.txInputs) == 0 {
		cp1, err := vm.Assemble("ADD 9 NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		args1 := [][]byte{{4}, {5}}

		cp2, err := vm.Assemble("ADD 13 NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		args2 := [][]byte{{6}, {7}}

		result.txInputs = []*types.TxInput{
			types.NewSpendInput(nil, *newHash(9), result.assetID, 10, 0, []byte{byte(vm.OP_TRUE)}),
			types.NewSpendInput(args1, *newHash(5), result.assetID, 20, 0, cp1),
			types.NewSpendInput(args2, *newHash(8), result.assetID, 40, 0, cp2),
		}
	}

	result.txInputs = append(result.txInputs, mockGasTxInput())

	if len(result.txOutputs) == 0 {
		cp1, err := vm.Assemble("ADD 17 NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}
		cp2, err := vm.Assemble("ADD 21 NUMEQUAL")
		if err != nil {
			tb.Fatal(err)
		}

		result.txOutputs = []*types.TxOutput{
			types.NewIntraChainOutput(result.assetID, 25, cp1),
			types.NewIntraChainOutput(result.assetID, 45, cp2),
		}
	}

	result.tx = &types.TxData{
		Version: result.txVersion,
		Inputs:  result.txInputs,
		Outputs: result.txOutputs,
	}

	return &result
}

func mockBlock() *bc.Block {
	return &bc.Block{
		BlockHeader: &bc.BlockHeader{
			Height: 666,
		},
	}
}

func mockGasTxInput() *types.TxInput {
	cp, _ := vmutil.DefaultCoinbaseProgram()
	return types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp)
}

// Like errors.Root, but also unwraps vm.Error objects.
func rootErr(e error) error {
	return errors.Root(e)
}

func hashData(data []byte) bc.Hash {
	var b32 [32]byte
	sha3pool.Sum256(b32[:], data)
	return bc.NewHash(b32)
}

func newHash(n byte) *bc.Hash {
	h := bc.NewHash([32]byte{n})
	return &h
}

func newAssetID(n byte) *bc.AssetID {
	a := bc.NewAssetID([32]byte{n})
	return &a
}

func txSpend(t *testing.T, tx *bc.Tx, index int) *bc.Spend {
	id := tx.InputIDs[index]
	res, err := tx.Spend(id)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func getMuxID(tx *bc.Tx) *bc.Hash {
	out := tx.Entries[*tx.ResultIds[0]]
	switch result := out.(type) {
	case *bc.IntraChainOutput:
		return result.Source.Ref
	case *bc.Retirement:
		return result.Source.Ref
	}
	return nil
}
