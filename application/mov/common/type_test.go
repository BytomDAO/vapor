package common

import (
	"testing"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/testutil"
)

func TestCalcUTXOHash(t *testing.T) {
	wantHash := "d94acbac0304e054569b0a2c2ab546be293552eb83d2d84af7234a013986a906"
	controlProgram := testutil.MustDecodeHexString("0014d6f0330717170c838e6ac4c643de61e4c035e9b7")
	sourceID := testutil.MustDecodeHash("3cada915465af2f08c93911bce7a100498fddb5738e5400269c4d5c2b2f5b261")
	order := Order{
		FromAssetID: consensus.BTMAssetID,
		Utxo: &MovUtxo{
			SourceID:       &sourceID,
			SourcePos:      1,
			Amount:         399551000,
			ControlProgram: controlProgram,
		},
	}

	if hash := order.UTXOHash(); hash.String() != wantHash {
		t.Fatal("The function is incorrect")
	}
}
