package types

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/vapor/encoding/blockchain"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/testutil"
)

func TestSerializationTxOutput(t *testing.T) {
	assetID := testutil.MustDecodeAsset("81756fdab39a17163b0ce582ee4ee256fb4d1e156c692b997d608a42ecb38d47")
	txOutput := NewIntraChainOutput(assetID, 254354, []byte("TestSerializationTxOutput"))
	wantHex := strings.Join([]string{
		"01", // asset version
		"40", // serialization length
		"00", // outType
		"3e", // output commitment length
		"81756fdab39a17163b0ce582ee4ee256fb4d1e156c692b997d608a42ecb38d47", // assetID
		"92c30f", // amount
		"01",     // version
		"19",     // control program length
		"5465737453657269616c697a6174696f6e54784f7574707574", // control program
		"00", // witness length
	}, "")

	// Test convert struct to hex
	var buffer bytes.Buffer
	if err := txOutput.writeTo(&buffer); err != nil {
		t.Fatal(err)
	}

	gotHex := hex.EncodeToString(buffer.Bytes())
	if gotHex != wantHex {
		t.Errorf("serialization bytes = %s want %s", gotHex, wantHex)
	}

	// Test convert hex to struct
	var gotTxOutput TxOutput
	decodeHex, err := hex.DecodeString(wantHex)
	if err != nil {
		t.Fatal(err)
	}

	if err := gotTxOutput.readFrom(blockchain.NewReader(decodeHex)); err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(*txOutput, gotTxOutput) {
		t.Errorf("expected marshaled/unmarshaled txoutput to be:\n%sgot:\n%s", spew.Sdump(*txOutput), spew.Sdump(gotTxOutput))
	}
}

func TestComputeOutputID(t *testing.T) {
	btmAssetID := testutil.MustDecodeAsset("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	cases := []struct {
		sc           *SpendCommitment
		wantOutputID string
	}{
		{
			sc: &SpendCommitment{
				AssetAmount:    bc.AssetAmount{AssetId: &btmAssetID, Amount: 1000},
				SourceID:       testutil.MustDecodeHash("4b5cb973f5bef4eadde4c89b92ee73312b940e84164da0594149554cc8a2adea"),
				SourcePosition: 2,
				VMVersion:      1,
				ControlProgram: testutil.MustDecodeHexString("0014cb9f2391bafe2bc1159b2c4c8a0f17ba1b4dd94e"),
			},
			wantOutputID: "73eea4d38b22ffd60fc30d0941f3875f45e29d424227bfde100193a08568605b",
		},
		{
			sc: &SpendCommitment{
				AssetAmount:    bc.AssetAmount{AssetId: &btmAssetID, Amount: 999},
				SourceID:       testutil.MustDecodeHash("9e74e35362ffc73c8967aa0008da8fcbc62a21d35673fb970445b5c2972f8603"),
				SourcePosition: 2,
				VMVersion:      1,
				ControlProgram: testutil.MustDecodeHexString("001418549d84daf53344d32563830c7cf979dc19d5c0"),
			},
			wantOutputID: "8371e76fd1c873503a326268bfd286ffe13009a0f1140d2c858e8187825696ab",
		},
	}

	for _, c := range cases {
		outputID, err := ComputeOutputID(c.sc)
		if err != nil {
			t.Fatal(err)
		}

		if c.wantOutputID != outputID.String() {
			t.Errorf("test compute output id fail, got:%s, want:%s", outputID.String(), c.wantOutputID)
		}
	}
}
