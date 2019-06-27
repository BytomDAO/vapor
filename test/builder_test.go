package test

import (
	"testing"

	acc "github.com/vapor/account"
	"github.com/vapor/blockchain/signers"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto/ed25519/chainkd"
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
