package test

import (
	"testing"

	acc "github.com/bytom/vapor/account"
	"github.com/bytom/vapor/blockchain/signers"
	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/crypto/ed25519/chainkd"
	"github.com/bytom/vapor/testutil"
)

var (
	//chainTxUtxoNum maximum utxo quantity in a tx
	chainTxUtxoNum = 5
)

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
			wantUtxo:   5,
		},
		{
			inputUtxo: []uint64{5, 4},
			wantInput: [][]uint64{
				[]uint64{5, 4},
			},
			wantOutput: [][]uint64{
				[]uint64{9},
			},
			wantUtxo: 9,
		},
		{
			inputUtxo: []uint64{5, 4, 1, 1},
			wantInput: [][]uint64{
				[]uint64{5, 4, 1, 1},
				[]uint64{1, 9},
			},
			wantOutput: [][]uint64{
				[]uint64{11},
				[]uint64{10},
			},
			wantUtxo: 11,
		},
		{
			inputUtxo: []uint64{22, 123, 53, 234, 23, 4, 2423, 24, 23, 43, 34, 234, 234, 24, 11, 16, 33, 59, 73, 89, 66},
			wantInput: [][]uint64{
				[]uint64{22, 123, 53, 234, 23, 4, 2423, 24, 23, 43, 34, 234, 234, 24, 11, 16, 33, 59, 73, 89},
				[]uint64{66, 3779},
			},
			wantOutput: [][]uint64{
				[]uint64{3779},
				[]uint64{3845},
			},
			wantUtxo: 3845,
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
				Amount:         amount,
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
				gotInput = append(gotInput, input.Amount())
			}

			gotOutput := []uint64{}
			for _, output := range tpl.Transaction.Outputs {
				gotOutput = append(gotOutput, output.AssetAmount().Amount)
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
