package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/testutil"
)

func TestTransaction(t *testing.T) {
	cases := []struct {
		tx   *Tx
		hex  string
		hash bc.Hash
	}{
		{
			tx: NewTx(TxData{
				Version:        1,
				SerializedSize: uint64(5),
				Inputs:         nil,
				Outputs:        nil,
			}),
			hex: strings.Join([]string{
				"07", // serflags
				"01", // transaction version
				"00", // tx time range
				"00", // inputs count
				"00", // outputs count
			}, ""),
			hash: testutil.MustDecodeHash("8e88b9cb4615128c7209dff695f68b8de5b38648bf3d44d2d0e6a674848539c9"),
		},
		{
			tx: NewTx(TxData{
				Version:        1,
				SerializedSize: uint64(112),
				Inputs: []*TxInput{
					NewCoinbaseInput([]byte("arbitrary")),
				},
				Outputs: []*TxOutput{
					NewIntraChainOutput(*consensus.BTMAssetID, 254354, []byte("true")),
					NewIntraChainOutput(*consensus.BTMAssetID, 254354, []byte("false")),
				},
			}),
			hex: strings.Join([]string{
				"07",                 // serflags
				"01",                 // transaction version
				"00",                 // tx time range
				"01",                 // inputs count
				"01",                 // input 0: asset version
				"0b",                 // input 0: input commitment length
				"02",                 // input 0: coinbase type flag
				"09",                 // input 0: arbitrary length
				"617262697472617279", // input 0: arbitrary data
				"00",                 // input 0: witness length
				"02",                 // outputs count
				"01",                 // output 0: asset version
				"2b",                 // output 0: serialization length
				"00",                 // output 0: outType
				"29",                 // output 0: output commitment length
				"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", // output 0: assetID
				"92c30f",   // output 0: amount
				"01",       // output 0: version
				"04",       // output 0: control program length
				"74727565", // output 0: control program
				"00",       // output 0: witness length
				"01",       // output 1: asset version
				"2c",       // output 1: serialization length
				"00",       // output 1: outType
				"2a",       // output 1: output commitment length
				"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", // output 1: assetID
				"92c30f",     // output 1: amount
				"01",         // output 1: version
				"05",         // output 1: control program length
				"66616c7365", // output 1: control program
				"00",         // output 1: witness length
			}, ""),
			hash: testutil.MustDecodeHash("2591a2af0d3690107215c2a47ab60c4e8d7547f04154ecd5ccab1db0d31e66b4"),
		},
	}
	for i, test := range cases {
		got := testutil.Serialize(t, test.tx)
		want, err := hex.DecodeString(test.hex)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(got, want) {
			t.Errorf("test %d: bytes = %x want %x", i, got, want)
		}
		if test.tx.ID != test.hash {
			t.Errorf("test %d: hash = %x want %x", i, test.tx.ID.Bytes(), test.hash.Bytes())
		}

		txJSON, err := json.Marshal(test.tx)
		if err != nil {
			t.Errorf("test %d: error marshaling tx to json: %s", i, err)
		}
		txFromJSON := Tx{}
		if err := json.Unmarshal(txJSON, &txFromJSON); err != nil {
			t.Errorf("test %d: error unmarshaling tx from json: %s", i, err)
		}
		if !testutil.DeepEqual(test.tx.TxData, txFromJSON.TxData) {
			t.Errorf("test %d: types.TxData -> json -> types.TxData: got:\n%s\nwant:\n%s", i, spew.Sdump(txFromJSON.TxData), spew.Sdump(test.tx.TxData))
		}

		tx1 := new(TxData)
		if err := tx1.UnmarshalText([]byte(test.hex)); err != nil {
			t.Errorf("test %d: unexpected err %v", i, err)
		}
		if !testutil.DeepEqual(*tx1, test.tx.TxData) {
			t.Errorf("test %d: tx1 is:\n%swant:\n%s", i, spew.Sdump(*tx1), spew.Sdump(test.tx.TxData))
		}
	}
}

func BenchmarkTxWriteToTrue(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, 0)
	}
}

func BenchmarkTxWriteToFalse(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, serRequired)
	}
}

func BenchmarkTxWriteToTrue200(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < 200; i++ {
		tx.Inputs = append(tx.Inputs, NewSpendInput(nil, bc.Hash{}, bc.AssetID{}, 0, 0, nil))
		tx.Outputs = append(tx.Outputs, NewIntraChainOutput(bc.AssetID{}, 0, nil))
	}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, 0)
	}
}

func BenchmarkTxWriteToFalse200(b *testing.B) {
	tx := &Tx{}
	for i := 0; i < 200; i++ {
		tx.Inputs = append(tx.Inputs, NewSpendInput(nil, bc.Hash{}, bc.AssetID{}, 0, 0, nil))
		tx.Outputs = append(tx.Outputs, NewIntraChainOutput(bc.AssetID{}, 0, nil))
	}
	for i := 0; i < b.N; i++ {
		tx.writeTo(ioutil.Discard, serRequired)
	}
}

func BenchmarkTxInputWriteToTrue(b *testing.B) {
	input := NewSpendInput(nil, bc.Hash{}, bc.AssetID{}, 0, 0, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		input.writeTo(ew)
	}
}

func BenchmarkTxInputWriteToFalse(b *testing.B) {
	input := NewSpendInput(nil, bc.Hash{}, bc.AssetID{}, 0, 0, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		input.writeTo(ew)
	}
}

func BenchmarkTxOutputWriteToTrue(b *testing.B) {
	output := NewIntraChainOutput(bc.AssetID{}, 0, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		output.writeTo(ew)
	}
}

func BenchmarkTxOutputWriteToFalse(b *testing.B) {
	output := NewIntraChainOutput(bc.AssetID{}, 0, nil)
	ew := errors.NewWriter(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		output.writeTo(ew)
	}
}

func BenchmarkAssetAmountWriteTo(b *testing.B) {
	aa := bc.AssetAmount{}
	for i := 0; i < b.N; i++ {
		aa.WriteTo(ioutil.Discard)
	}
}
