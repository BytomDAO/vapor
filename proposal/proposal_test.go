package proposal

import (
	"bytes"
	"testing"

	"github.com/vapor/protocol/bc"
)

func TestCreateCoinbaseTx(t *testing.T) {
	cases := []struct {
		desc          string
		blockHeight   uint64
		wantArbitrary []byte
		wantAmount    uint64
	}{
		{
			desc:          "the coinbase block height is reductionInterval",
			blockHeight:   1,
			wantArbitrary: []byte{0x00, 0x31},
			wantAmount:    0,
		},
		{
			desc:          "the coinbase block height is reductionInterval",
			blockHeight:   100,
			wantArbitrary: []byte{0x00, 0x31, 0x30, 0x30},
			wantAmount:    0,
		},
	}

	for i, c := range cases {
		coinbaseTx, err := createCoinbaseTx(nil, c.blockHeight)
		if err != nil {
			t.Fatal(err)
		}

		input, _ := coinbaseTx.Entries[coinbaseTx.Tx.InputIDs[0]].(*bc.Coinbase)
		gotArbitrary := input.Arbitrary
		if res := bytes.Compare(gotArbitrary, c.wantArbitrary); res != 0 {
			t.Fatalf("coinbase tx arbitrary dismatch, case: %d, got: %d, want: %d", i, gotArbitrary, c.wantArbitrary)
		}

		gotAmount := coinbaseTx.Outputs[0].AssetAmount().Amount
		if gotAmount != c.wantAmount {
			t.Fatalf("coinbase tx output amount dismatch, case: %d, got: %d, want: %d", i, gotAmount, c.wantAmount)
		}
	}
}
