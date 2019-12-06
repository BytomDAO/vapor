package account

import (
	"testing"
	"time"

	"github.com/bytom/vapor/blockchain/txbuilder"
	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/testutil"
)

func TestMergeSpendAction(t *testing.T) {
	testBTM := &bc.AssetID{}
	if err := testBTM.UnmarshalText([]byte("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")); err != nil {
		t.Fatal(err)
	}

	testAssetID1 := &bc.AssetID{}
	if err := testAssetID1.UnmarshalText([]byte("50ec80b6bc48073f6aa8fa045131a71213c33f3681203b15ddc2e4b81f1f4730")); err != nil {
		t.Fatal(err)
	}

	testAssetID2 := &bc.AssetID{}
	if err := testAssetID2.UnmarshalText([]byte("43c6946d092b2959c1a82e90b282c68fca63e66de289048f6acd6cea9383c79c")); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		testActions     []txbuilder.Action
		wantActions     []txbuilder.Action
		testActionCount int
		wantActionCount int
	}{
		{
			testActions: []txbuilder.Action{
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  200,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
			},
			wantActions: []txbuilder.Action{
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  600,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
			},
			testActionCount: 4,
			wantActionCount: 2,
		},
		{
			testActions: []txbuilder.Action{
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  200,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  500,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
			},
			wantActions: []txbuilder.Action{
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  600,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  500,
					},
					AccountID: "test_account",
				}),
			},
			testActionCount: 4,
			wantActionCount: 2,
		},
		{
			testActions: []txbuilder.Action{
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  200,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID2,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID2,
						Amount:  500,
					},
					AccountID: "test_account",
				}),
			},
			wantActions: []txbuilder.Action{
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  500,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID2,
						Amount:  800,
					},
					AccountID: "test_account",
				}),
			},
			testActionCount: 5,
			wantActionCount: 3,
		},
		{
			testActions: []txbuilder.Action{
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  200,
					},
					AccountID: "test_account1",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  500,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  300,
					},
					AccountID: "test_account1",
				}),
			},
			wantActions: []txbuilder.Action{
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  600,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  200,
					},
					AccountID: "test_account1",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  300,
					},
					AccountID: "test_account1",
				}),
			},
			testActionCount: 4,
			wantActionCount: 3,
		},
		{
			testActions: []txbuilder.Action{
				txbuilder.Action(&spendUTXOAction{
					OutputID: &bc.Hash{V0: 128},
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  200,
					},
					AccountID: "test_account1",
				}),
				txbuilder.Action(&spendUTXOAction{
					OutputID: &bc.Hash{V0: 256},
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID2,
						Amount:  300,
					},
					AccountID: "test_account2",
				}),
			},
			wantActions: []txbuilder.Action{
				txbuilder.Action(&spendUTXOAction{
					OutputID: &bc.Hash{V0: 128},
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  200,
					},
					AccountID: "test_account1",
				}),
				txbuilder.Action(&spendUTXOAction{
					OutputID: &bc.Hash{V0: 256},
				}),
				txbuilder.Action(&spendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID2,
						Amount:  300,
					},
					AccountID: "test_account2",
				}),
			},
			testActionCount: 5,
			wantActionCount: 5,
		},
	}

	for _, c := range cases {
		gotActions := MergeSpendAction(c.testActions)

		gotMap := make(map[string]uint64)
		wantMap := make(map[string]uint64)
		for _, got := range gotActions {
			switch got := got.(type) {
			case *spendAction:
				gotKey := got.AssetId.String() + got.AccountID
				gotMap[gotKey] = got.Amount
			default:
				continue
			}
		}

		for _, want := range c.wantActions {
			switch want := want.(type) {
			case *spendAction:
				wantKey := want.AssetId.String() + want.AccountID
				wantMap[wantKey] = want.Amount
			default:
				continue
			}
		}

		for key := range gotMap {
			if gotMap[key] != wantMap[key] {
				t.Fatalf("gotMap[%s]=%v, wantMap[%s]=%v", key, gotMap[key], key, wantMap[key])
			}
		}

		if len(gotActions) != c.wantActionCount {
			t.Fatalf("number of gotActions=%d, wantActions=%d", len(gotActions), c.wantActionCount)
		}
	}
}

func TestReserveBtmUtxoChain(t *testing.T) {
	chainTxUtxoNum = 3
	chainTxMergeGas = uint64(10000000)

	utxos := []*UTXO{}
	m := mockAccountManager(t)
	for i := uint64(1); i <= 20; i++ {
		utxo := &UTXO{
			OutputID:  bc.Hash{V0: i},
			AccountID: "TestAccountID",
			AssetID:   *consensus.BTMAssetID,
			Amount:    i * chainTxMergeGas,
		}
		utxos = append(utxos, utxo)

		m.store.SetStandardUTXO(utxo.OutputID, utxo)
	}

	cases := []struct {
		amount uint64
		want   []uint64
		err    bool
	}{
		{
			amount: 1 * chainTxMergeGas,
			want:   []uint64{1},
		},
		{
			amount: 888888 * chainTxMergeGas,
			want:   []uint64{},
			err:    true,
		},
		{
			amount: 7 * chainTxMergeGas,
			want:   []uint64{4, 3, 1},
		},
		{
			amount: 15 * chainTxMergeGas,
			want:   []uint64{5, 4, 3, 2, 1, 6},
		},
		{
			amount: 163 * chainTxMergeGas,
			want:   []uint64{20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 2, 1, 3},
		},
	}

	for i, c := range cases {
		m.utxoKeeper.expireReservation(time.Unix(999999999, 0))
		utxos, err := m.reserveBtmUtxoChain(&txbuilder.TemplateBuilder{}, "TestAccountID", c.amount, false)

		if err != nil != c.err {
			t.Fatalf("case %d got err %v want err = %v", i, err, c.err)
		}

		got := []uint64{}
		for _, utxo := range utxos {
			got = append(got, utxo.Amount/chainTxMergeGas)
		}

		if !testutil.DeepEqual(got, c.want) {
			t.Fatalf("case %d got %d want %d", i, got, c.want)
		}
	}
}
