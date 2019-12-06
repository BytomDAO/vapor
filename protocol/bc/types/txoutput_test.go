package types

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/vapor/encoding/blockchain"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/testutil"
)

func TestSerializationIntraChainTxOutput(t *testing.T) {
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

func TestSerializationCrossChainTxOutput(t *testing.T) {
	assetID := testutil.MustDecodeAsset("81756fdab39a17163b0ce582ee4ee256fb4d1e156c692b997d608a42ecb38d47")
	txOutput := NewCrossChainOutput(assetID, 254354, []byte("TestSerializationTxOutput"))
	wantHex := strings.Join([]string{
		"01", // asset version
		"40", // serialization length
		"01", // outType
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

func TestSerializationVoteOutput(t *testing.T) {
	assetID := testutil.MustDecodeAsset("81756fdab39a17163b0ce582ee4ee256fb4d1e156c692b997d608a42ecb38d47")
	voteTxOutput := NewVoteOutput(assetID, 1000, []byte("TestSerializationTxOutput"), []byte("af594006a40837d9f028daabb6d589df0b9138daefad5683e5233c2646279217294a8d532e60863bcf196625a35fb8ceeffa3c09610eb92dcfb655a947f13269"))

	wantHex := strings.Join([]string{
		"01",   // asset version
		"c101", // serialization length
		"02",   // outType
		"8001", // output xpub length
		"6166353934303036613430383337643966303238646161626236643538396466306239313338646165666164353638336535323333633236343632373932313732393461386435333265363038363362636631393636323561333566623863656566666133633039363130656239326463666236353561393437663133323639", // xpub
		"3d", // output commitment length
		"81756fdab39a17163b0ce582ee4ee256fb4d1e156c692b997d608a42ecb38d47", // assetID
		"e807", // amount
		"01",   // version
		"19",   // control program length
		"5465737453657269616c697a6174696f6e54784f7574707574", // control program
		"00", // witness length
	}, "")

	// Test convert struct to hex
	var buffer bytes.Buffer
	if err := voteTxOutput.writeTo(&buffer); err != nil {
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

	if !testutil.DeepEqual(*voteTxOutput, gotTxOutput) {
		t.Errorf("expected marshaled/unmarshaled txoutput to be:\n%sgot:\n%s", spew.Sdump(*voteTxOutput), spew.Sdump(gotTxOutput))
	}
}

func TestComputeOutputID(t *testing.T) {
	btmAssetID := testutil.MustDecodeAsset("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	cases := []struct {
		sc           *SpendCommitment
		inputType    uint8
		vote         []byte
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
			inputType:    SpendInputType,
			vote:         nil,
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
			inputType:    SpendInputType,
			vote:         nil,
			wantOutputID: "8371e76fd1c873503a326268bfd286ffe13009a0f1140d2c858e8187825696ab",
		},
		{
			sc: &SpendCommitment{
				AssetAmount:    bc.AssetAmount{AssetId: &btmAssetID, Amount: 999},
				SourceID:       testutil.MustDecodeHash("993d3797fa3b2d958f300e599987dc10904b13f56ce89d158f60f9131424e0e2"),
				SourcePosition: 2,
				VMVersion:      1,
				ControlProgram: testutil.MustDecodeHexString("00145c47f3a0dd3e1e9956fe5b0f897072ed33f9efb9"),
			},
			inputType:    VetoInputType,
			vote:         []byte("af594006a40837d9f028daabb6d589df0b9138daefad5683e5233c2646279217294a8d532e60863bcf196625a35fb8ceeffa3c09610eb92dcfb655a947f13269"),
			wantOutputID: "a4de5a81babc7949d6b38d1cd4bcbc83da340387e747b5f521af3e427c6b0132",
		},
		{
			sc: &SpendCommitment{
				AssetAmount:    bc.AssetAmount{AssetId: &btmAssetID, Amount: 999},
				SourceID:       testutil.MustDecodeHash("993d3797fa3b2d958f300e599987dc10904b13f56ce89d158f60f9131424e0e2"),
				SourcePosition: 2,
				VMVersion:      1,
				ControlProgram: testutil.MustDecodeHexString("00145c47f3a0dd3e1e9956fe5b0f897072ed33f9efb9"),
			},
			inputType:    VetoInputType,
			vote:         []byte(""),
			wantOutputID: "e42a48ef401b993c5e523b6a7b5456ad4b297c7aeda163405f265d8d00af983e",
		},
	}

	for _, c := range cases {
		outputID, err := ComputeOutputID(c.sc, c.inputType, c.vote)
		if err != nil {
			t.Fatal(err)
		}

		if c.wantOutputID != outputID.String() {
			t.Errorf("test compute output id fail, got:%s, want:%s", outputID.String(), c.wantOutputID)
		}
	}
}
