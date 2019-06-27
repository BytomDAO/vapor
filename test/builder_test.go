package test

import (
	"context"
	"testing"

	acc "github.com/vapor/account"
	"github.com/vapor/blockchain/signers"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/testutil"
)

var (
	//chainTxUtxoNum maximum utxo quantity in a tx
	chainTxUtxoNum = 5
	//chainTxMergeGas chain tx gas
	chainTxMergeGas = uint64(10000000)
)

// func TestReserveBtmUtxoChain(t *testing.T) {
// 	dirPath, err := ioutil.TempDir(".", "")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer os.RemoveAll(dirPath)
// 	testDB := dbm.NewDB("testdb", "memdb", dirPath)
// 	store := database.NewAccountStore(testDB)
// 	chainTxUtxoNum = 3
// 	utxos := []*acc.UTXO{}
// 	m := mockAccountManager(t)
// 	for i := uint64(1); i <= 20; i++ {
// 		utxo := &acc.UTXO{
// 			OutputID:  bc.Hash{V0: i},
// 			AccountID: "TestAccountID",
// 			AssetID:   *consensus.BTMAssetID,
// 			Amount:    i * chainTxMergeGas,
// 		}
// 		utxos = append(utxos, utxo)

// 		data, err := json.Marshal(utxo)
// 		if err != nil {
// 			t.Fatal(err)
// 		}

// 		store.SetStandardUTXO(utxo.OutputID, data)
// 	}

// 	cases := []struct {
// 		amount uint64
// 		want   []uint64
// 		err    bool
// 	}{
// 		{
// 			amount: 1 * chainTxMergeGas,
// 			want:   []uint64{1},
// 		},
// 		{
// 			amount: 888888 * chainTxMergeGas,
// 			want:   []uint64{},
// 			err:    true,
// 		},
// 		{
// 			amount: 7 * chainTxMergeGas,
// 			want:   []uint64{4, 3, 1},
// 		},
// 		{
// 			amount: 15 * chainTxMergeGas,
// 			want:   []uint64{5, 4, 3, 2, 1, 6},
// 		},
// 		{
// 			amount: 163 * chainTxMergeGas,
// 			want:   []uint64{20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 2, 1, 3},
// 		},
// 	}

// 	for i, c := range cases {
// 		m.utxoKeeper.expireReservation(time.Unix(999999999, 0))
// 		utxos, err := m.Manager.ReserveBtmUtxoChain(&txbuilder.TemplateBuilder{}, "TestAccountID", c.amount, false)

// 		if err != nil != c.err {
// 			t.Fatalf("case %d got err %v want err = %v", i, err, c.err)
// 		}

// 		got := []uint64{}
// 		for _, utxo := range utxos {
// 			got = append(got, utxo.Amount/chainTxMergeGas)
// 		}

// 		if !testutil.DeepEqual(got, c.want) {
// 			t.Fatalf("case %d got %d want %d", i, got, c.want)
// 		}
// 	}
// }

func TestBuildBtmTxChain(t *testing.T) {
	chainTxUtxoNum = 3
	m := mockAccountManager(t)
	cases := []struct {
		inputUtxo  []uint64
		wantInput  [][]uint64
		wantOutput [][]uint64
		wantUtxo   uint64
	}{
		{
			inputUtxo:  []uint64{5},
			wantInput:  [][]uint64{},
			wantOutput: [][]uint64{},
			wantUtxo:   5 * chainTxMergeGas,
		},
		{
			inputUtxo: []uint64{5, 4},
			wantInput: [][]uint64{
				[]uint64{5, 4},
			},
			wantOutput: [][]uint64{
				[]uint64{8},
			},
			wantUtxo: 8 * chainTxMergeGas,
		},
		{
			inputUtxo: []uint64{5, 4, 1, 1},
			wantInput: [][]uint64{
				[]uint64{5, 4, 1, 1},
				[]uint64{1, 9},
			},
			wantOutput: [][]uint64{
				[]uint64{10},
				[]uint64{9},
			},
			wantUtxo: 10 * chainTxMergeGas,
		},
		{
			inputUtxo: []uint64{22, 123, 53, 234, 23, 4, 2423, 24, 23, 43, 34, 234, 234, 24},
			wantInput: [][]uint64{
				[]uint64{22, 123, 53, 234, 23},
				[]uint64{4, 2423, 24, 23, 43},
				[]uint64{34, 234, 234, 24, 454},
				[]uint64{2516, 979},
				[]uint64{234, 24, 197},
				[]uint64{260, 2469, 310},
				[]uint64{454, 3038},
			},
			wantOutput: [][]uint64{
				[]uint64{454},
				[]uint64{2516},
				[]uint64{979},
				[]uint64{3494},
				[]uint64{454},
				[]uint64{3038},
				[]uint64{3491},
			},
			wantUtxo: 3494 * chainTxMergeGas,
		},
	}

	acct, err := m.Manager.Create([]chainkd.XPub{testutil.TestXPub}, 1, "testAccount", signers.BIP0044)
	if err != nil {
		t.Fatal(err)
	}

	acp, err := m.Manager.CreateAddress(acct.ID, false)
	if err != nil {
		t.Fatal(err)
	}

	for caseIndex, c := range cases {
		utxos := []*acc.UTXO{}
		for _, amount := range c.inputUtxo {
			utxos = append(utxos, &acc.UTXO{
				Amount:         amount * chainTxMergeGas,
				AssetID:        *consensus.BTMAssetID,
				Address:        acp.Address,
				ControlProgram: acp.ControlProgram,
			})
		}

		tpls, gotUtxo, err := m.Manager.BuildBtmTxChain(utxos, acct.Signer)
		if err != nil {
			t.Fatal(err)
		}

		for i, tpl := range tpls {
			gotInput := []uint64{}
			for _, input := range tpl.Transaction.Inputs {
				gotInput = append(gotInput, input.Amount()/chainTxMergeGas)
			}

			gotOutput := []uint64{}
			for _, output := range tpl.Transaction.Outputs {
				gotOutput = append(gotOutput, output.AssetAmount().Amount/chainTxMergeGas)
			}

			if !testutil.DeepEqual(c.wantInput[i], gotInput) {
				t.Errorf("case %d tx %d input got %d want %d", caseIndex, i, gotInput, c.wantInput[i])
			}
			if !testutil.DeepEqual(c.wantOutput[i], gotOutput) {
				t.Errorf("case %d tx %d output got %d want %d", caseIndex, i, gotOutput, c.wantOutput[i])
			}
		}

		if c.wantUtxo != gotUtxo.Amount {
			t.Errorf("case %d got utxo=%d want utxo=%d", caseIndex, gotUtxo.Amount, c.wantUtxo)
		}
	}
}

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
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  200,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
			},
			wantActions: []txbuilder.Action{
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  600,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
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
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  200,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  500,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
			},
			wantActions: []txbuilder.Action{
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  600,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
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
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  200,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID2,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  300,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID2,
						Amount:  500,
					},
					AccountID: "test_account",
				}),
			},
			wantActions: []txbuilder.Action{
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  500,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
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
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  200,
					},
					AccountID: "test_account1",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  500,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  300,
					},
					AccountID: "test_account1",
				}),
			},
			wantActions: []txbuilder.Action{
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  600,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  200,
					},
					AccountID: "test_account1",
				}),
				txbuilder.Action(&mockSpendAction{
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
				txbuilder.Action(&mockSpendUTXOAction{
					OutputID: &bc.Hash{V0: 128},
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  200,
					},
					AccountID: "test_account1",
				}),
				txbuilder.Action(&mockSpendUTXOAction{
					OutputID: &bc.Hash{V0: 256},
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID2,
						Amount:  300,
					},
					AccountID: "test_account2",
				}),
			},
			wantActions: []txbuilder.Action{
				txbuilder.Action(&mockSpendUTXOAction{
					OutputID: &bc.Hash{V0: 128},
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testBTM,
						Amount:  100,
					},
					AccountID: "test_account",
				}),
				txbuilder.Action(&mockSpendAction{
					AssetAmount: bc.AssetAmount{
						AssetId: testAssetID1,
						Amount:  200,
					},
					AccountID: "test_account1",
				}),
				txbuilder.Action(&mockSpendUTXOAction{
					OutputID: &bc.Hash{V0: 256},
				}),
				txbuilder.Action(&mockSpendAction{
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

	for i, c := range cases {
		gotActions := acc.MergeSpendAction(c.testActions)

		gotMap := make(map[string]uint64)
		wantMap := make(map[string]uint64)
		for _, got := range gotActions {
			switch got := got.(type) {
			case *mockSpendAction:
				gotKey := got.AssetId.String() + got.AccountID
				gotMap[gotKey] = got.Amount
			default:
				continue
			}
		}

		for _, want := range c.wantActions {
			switch want := want.(type) {
			case *mockSpendAction:
				wantKey := want.AssetId.String() + want.AccountID
				wantMap[wantKey] = want.Amount
			default:
				continue
			}
		}

		for key := range gotMap {
			if gotMap[key] != wantMap[key] {
				t.Errorf("case: %v, gotMap[%s]=%v, wantMap[%s]=%v", i, key, gotMap[key], key, wantMap[key])
			}
		}

		if len(gotActions) != c.wantActionCount {
			t.Errorf("case: %v, number of gotActions=%d, wantActions=%d", i, len(gotActions), c.wantActionCount)
		}
	}
}

func TestCalcMergeGas(t *testing.T) {
	chainTxUtxoNum = 10
	cases := []struct {
		utxoNum int
		gas     uint64
	}{
		{
			utxoNum: 0,
			gas:     0,
		},
		{
			utxoNum: 1,
			gas:     0,
		},
		{
			utxoNum: 9,
			gas:     chainTxMergeGas * 2,
		},
		{
			utxoNum: 10,
			gas:     chainTxMergeGas * 3,
		},
		{
			utxoNum: 11,
			gas:     chainTxMergeGas * 3,
		},
		{
			utxoNum: 20,
			gas:     chainTxMergeGas * 5,
		},
		{
			utxoNum: 21,
			gas:     chainTxMergeGas * 5,
		},
		{
			utxoNum: 74,
			gas:     chainTxMergeGas * 19,
		},
	}

	for i, c := range cases {
		gas := acc.CalcMergeGas(c.utxoNum)
		if gas != c.gas {
			t.Fatalf("case %d got %d want %d", i, gas, c.gas)
		}
	}
}

type mockSpendAction struct {
	accounts *acc.Manager
	bc.AssetAmount
	AccountID      string `json:"account_id"`
	UseUnconfirmed bool   `json:"use_unconfirmed"`
}

func (a *mockSpendAction) ActionType() string {
	return "spend_account"
}

func (a *mockSpendAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	// var missing []string
	// if a.AccountID == "" {
	// 	missing = append(missing, "account_id")
	// }
	// if a.AssetId.IsZero() {
	// 	missing = append(missing, "asset_id")
	// }
	// if len(missing) > 0 {
	// 	return txbuilder.MissingFieldsError(missing...)
	// }

	// acct, err := a.accounts.FindByID(a.AccountID)
	// if err != nil {
	// 	return errors.Wrap(err, "get account info")
	// }

	// res, err := a.accounts.utxoKeeper.Reserve(a.AccountID, a.AssetId, a.Amount, a.UseUnconfirmed, nil, b.MaxTime())
	// if err != nil {
	// 	return errors.Wrap(err, "reserving utxos")
	// }

	// // Cancel the reservation if the build gets rolled back.
	// b.OnRollback(func() { a.accounts.utxoKeeper.Cancel(res.id) })
	// for _, r := range res.utxos {
	// 	txInput, sigInst, err := UtxoToInputs(acct.Signer, r)
	// 	if err != nil {
	// 		return errors.Wrap(err, "creating inputs")
	// 	}

	// 	if err = b.AddInput(txInput, sigInst); err != nil {
	// 		return errors.Wrap(err, "adding inputs")
	// 	}
	// }

	// if res.change > 0 {
	// 	acp, err := a.accounts.CreateAddress(a.AccountID, true)
	// 	if err != nil {
	// 		return errors.Wrap(err, "creating control program")
	// 	}

	// 	// Don't insert the control program until callbacks are executed.
	// 	a.accounts.insertControlProgramDelayed(b, acp)
	// 	if err = b.AddOutput(types.NewIntraChainOutput(*a.AssetId, res.change, acp.ControlProgram)); err != nil {
	// 		return errors.Wrap(err, "adding change output")
	// 	}
	// }
	return nil
}

type mockSpendUTXOAction struct {
	accounts       *acc.Manager
	OutputID       *bc.Hash                     `json:"output_id"`
	UseUnconfirmed bool                         `json:"use_unconfirmed"`
	Arguments      []txbuilder.ContractArgument `json:"arguments"`
}

func (a *mockSpendUTXOAction) ActionType() string {
	return "spend_account_unspent_output"
}

func (a *mockSpendUTXOAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	// if a.OutputID == nil {
	// 	return txbuilder.MissingFieldsError("output_id")
	// }

	// res, err := a.accounts.utxoKeeper.ReserveParticular(*a.OutputID, a.UseUnconfirmed, b.MaxTime())
	// if err != nil {
	// 	return err
	// }

	// b.OnRollback(func() { a.accounts.utxoKeeper.Cancel(res.id) })
	// var accountSigner *signers.Signer
	// if len(res.utxos[0].AccountID) != 0 {
	// 	account, err := a.accounts.FindByID(res.utxos[0].AccountID)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	accountSigner = account.Signers
	// }

	// txInput, sigInst, err := UtxoToInputs(accountSigner, res.utxos[0])
	// if err != nil {
	// 	return err
	// }

	// if a.Arguments == nil {
	// 	return b.AddInput(txInput, sigInst)
	// }

	// sigInst = &txbuilder.SigningInstruction{}
	// if err := txbuilder.AddContractArgs(sigInst, a.Arguments); err != nil {
	// 	return err
	// }

	// return b.AddInput(txInput, sigInst)

	return nil
}
