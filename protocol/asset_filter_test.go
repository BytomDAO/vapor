package protocol

import (
	"testing"

	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/testutil"
)

func TestIsDust(t *testing.T) {
	assetFilter := NewAssetFilter("184e1cc4ee4845023888810a79eed7a42c02c544cf2c61ceac05e176d575bd46")
	cases := []struct {
		tx         *types.Tx
		wantIsDust bool
	}{
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewCrossChainInput(nil, bc.Hash{}, testutil.MustDecodeAsset("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"), 1e8, 1, 1, []byte("{\n  \"decimals\": 8,\n  \"description\": \"Bytom Official Issue\",\n  \"name\": \"BTM\",\n  \"symbol\": \"BTM\"\n}"), []byte("assetbtm"))},
				Outputs: []*types.TxOutput{
					types.NewIntraChainOutput(testutil.MustDecodeAsset("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"), 1e8, []byte{0x51}),
				},
			}),
			wantIsDust: false,
		},
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewCrossChainInput(nil, bc.Hash{}, testutil.MustDecodeAsset("184e1cc4ee4845023888810a79eed7a42c02c544cf2c61ceac05e176d575bd46"), 1e8, 1, 1, []byte("{\n  \"decimals\": 6,\n  \"description\": {\n    \"issue_asset_action\": \"open_federation_cross_chain\"\n  },\n  \"name\": \"USDT\",\n  \"quorum\": \"3\",\n  \"reissue\": \"true\",\n  \"symbol\": \"USDT\"\n}"), []byte("assetusdt"))},
				Outputs: []*types.TxOutput{
					types.NewIntraChainOutput(testutil.MustDecodeAsset("184e1cc4ee4845023888810a79eed7a42c02c544cf2c61ceac05e176d575bd46"), 1e8, []byte{0x51}),
				},
			}),
			wantIsDust: false,
		},
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewCrossChainInput(nil, bc.Hash{}, testutil.MustDecodeAsset("47fcd4d7c22d1d38931a6cd7767156babbd5f05bbbb3f7d3900635b56eb1b67e"), 1e8, 1, 1, []byte("{\n  \"decimals\": 8,\n  \"description\": {},\n  \"name\": \"SUP\",\n  \"quorum\": 1,\n  \"reissue\": \"false\",\n  \"symbol\": \"SUP\"\n}"), []byte("assetsup"))},
				Outputs: []*types.TxOutput{
					types.NewIntraChainOutput(testutil.MustDecodeAsset("47fcd4d7c22d1d38931a6cd7767156babbd5f05bbbb3f7d3900635b56eb1b67e"), 1e8, []byte{0x51}),
				},
			}),
			wantIsDust: false,
		},
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewCrossChainInput(nil, bc.Hash{}, testutil.MustDecodeAsset("c4644dd6643475d57ed624f63129ab815f282b61f4bb07646d73423a6e1a1563"), 1e8, 1, 1, []byte("{\n\"decimals\":6,\n\"description\":{\n\"issue_asset_action\":\"open_federation_cross_chain\"\n},\n\"name\":\"USDC\",\n\"quorum\":\"3\",\n\"reissue\":\"true\",\n\"symbol\":\"USDC\"\n}"), []byte("assetusdc"))},
				Outputs: []*types.TxOutput{
					types.NewIntraChainOutput(testutil.MustDecodeAsset("c4644dd6643475d57ed624f63129ab815f282b61f4bb07646d73423a6e1a1563"), 1e8, []byte{0x51}),
				},
			}),
			wantIsDust: true,
		},
	}

	for i, c := range cases {
		if gotIsDust := assetFilter.IsDust(c.tx); gotIsDust != c.wantIsDust {
			t.Errorf("case %d: fail on AssetFilter TestIsDust", i)
		}
	}
}
