package service

import (
	"encoding/json"
	"testing"

	"github.com/vapor/consensus"
	"github.com/vapor/protocol/bc"
)

func TestBuildRequest(t *testing.T) {
	cases := []struct {
		input   InputAction
		outputs []OutputAction
		want    string
		err     error
	}{
		{
			input: InputAction{
				Type:      "spend_account",
				AccountID: "9bb77612-350e-4d53-81e2-525b28247ba5",
				AssetAmount: bc.AssetAmount{
					AssetId: consensus.BTMAssetID,
					Amount:  100,
				},
			},
			outputs: []OutputAction{
				OutputAction{
					Type:    "control_address",
					Address: "sp1qlryy65a5apylphqp6axvhx7nd6y2zlexuvn7gf",
					AssetAmount: bc.AssetAmount{
						AssetId: consensus.BTMAssetID,
						Amount:  100,
					},
				},
			},
			want: `{"actions":[{"type":"spend_account","account_id":"9bb77612-350e-4d53-81e2-525b28247ba5","asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount":100},{"type":"control_address","address":"sp1qlryy65a5apylphqp6axvhx7nd6y2zlexuvn7gf","asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount":100}]}`,
		},
		{
			input: InputAction{
				Type:      "spend_account",
				AccountID: "9bb77612-350e-4d53-81e2-525b28247ba5",
				AssetAmount: bc.AssetAmount{
					AssetId: consensus.BTMAssetID,
					Amount:  100,
				},
			},
			outputs: []OutputAction{
				OutputAction{
					Type:    "control_address",
					Address: "sp1qlryy65a5apylphqp6axvhx7nd6y2zlexuvn7gf",
					AssetAmount: bc.AssetAmount{
						AssetId: consensus.BTMAssetID,
						Amount:  50,
					},
				},
				OutputAction{
					Type:    "control_address",
					Address: "sp1qklmexrd32ch8yc8xhkpkdx05wye75pvzuy2gch",
					AssetAmount: bc.AssetAmount{
						AssetId: consensus.BTMAssetID,
						Amount:  50,
					},
				},
			},
			want: `{"actions":[{"type":"spend_account","account_id":"9bb77612-350e-4d53-81e2-525b28247ba5","asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount":100},{"type":"control_address","address":"sp1qklmexrd32ch8yc8xhkpkdx05wye75pvzuy2gch","asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount":50},{"type":"control_address","address":"sp1qklmexrd32ch8yc8xhkpkdx05wye75pvzuy2gch","asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount":50}]}`,
		},
	}

	for i, c := range cases {
		n := &Node{}
		req := &buildSpendReq{}
		if err := n.buildRequest(c.input, c.outputs, req); err != nil {
			t.Fatal(err)
		}

		buildReq, err := json.Marshal(req)
		if err != nil {
			t.Fatal(err)
		}

		if string(buildReq) != string(c.want) {
			t.Fatal(i, string(buildReq))
		}

	}
}
