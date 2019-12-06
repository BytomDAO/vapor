package wallet

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/bytom/vapor/account"
	"github.com/bytom/vapor/consensus"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/testutil"
)

func TestGetAccountUtxos(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	testStore := NewMockWalletStore(testDB)
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	cases := []struct {
		dbUtxos          map[string]*account.UTXO
		unconfirmedUtxos []*account.UTXO
		id               string
		unconfirmed      bool
		isSmartContract  bool
		wantUtxos        []*account.UTXO
	}{
		{
			dbUtxos:         map[string]*account.UTXO{},
			id:              "",
			unconfirmed:     false,
			isSmartContract: false,
			wantUtxos:       []*account.UTXO{},
		},
		{
			dbUtxos: map[string]*account.UTXO{
				string(StandardUTXOKey(bc.Hash{V0: 1})): &account.UTXO{
					OutputID: bc.Hash{V0: 1},
				},
				string(StandardUTXOKey(bc.Hash{V0: 2})): &account.UTXO{
					OutputID: bc.Hash{V0: 2},
				},
				string(StandardUTXOKey(bc.Hash{V0: 3})): &account.UTXO{
					OutputID: bc.Hash{V0: 3},
				},
				string(ContractUTXOKey(bc.Hash{V0: 4})): &account.UTXO{
					OutputID: bc.Hash{V0: 4},
				},
			},
			unconfirmedUtxos: []*account.UTXO{},
			id:               "",
			isSmartContract:  false,
			wantUtxos: []*account.UTXO{
				&account.UTXO{OutputID: bc.Hash{V0: 1}},
				&account.UTXO{OutputID: bc.Hash{V0: 2}},
				&account.UTXO{OutputID: bc.Hash{V0: 3}},
			},
		},
		{
			dbUtxos: map[string]*account.UTXO{
				string(StandardUTXOKey(bc.Hash{V0: 1})): &account.UTXO{
					OutputID: bc.Hash{V0: 1},
				},
				string(StandardUTXOKey(bc.Hash{V0: 2})): &account.UTXO{
					OutputID: bc.Hash{V0: 2},
				},
				string(StandardUTXOKey(bc.Hash{V0: 3})): &account.UTXO{
					OutputID: bc.Hash{V0: 3},
				},
				string(ContractUTXOKey(bc.Hash{V0: 4})): &account.UTXO{
					OutputID: bc.Hash{V0: 4},
				},
			},
			unconfirmedUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 5},
					ControlProgram: []byte("smart contract"),
				},
			},
			id:              "",
			unconfirmed:     false,
			isSmartContract: true,
			wantUtxos: []*account.UTXO{
				&account.UTXO{OutputID: bc.Hash{V0: 4}},
			},
		},
		{
			dbUtxos: map[string]*account.UTXO{
				string(StandardUTXOKey(bc.Hash{V0: 1})): &account.UTXO{
					OutputID: bc.Hash{V0: 1},
				},
				string(StandardUTXOKey(bc.Hash{V0: 1, V1: 2})): &account.UTXO{
					OutputID: bc.Hash{V0: 1, V1: 2},
				},
				string(StandardUTXOKey(bc.Hash{V0: 2})): &account.UTXO{
					OutputID: bc.Hash{V0: 2},
				},
				string(StandardUTXOKey(bc.Hash{V0: 2, V1: 2})): &account.UTXO{
					OutputID: bc.Hash{V0: 2, V1: 2},
				},
			},
			unconfirmedUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 6},
					ControlProgram: []byte{0x51},
				},
			},
			id:              "0000000000000002",
			unconfirmed:     false,
			isSmartContract: false,
			wantUtxos: []*account.UTXO{
				&account.UTXO{OutputID: bc.Hash{V0: 2}},
				&account.UTXO{OutputID: bc.Hash{V0: 2, V1: 2}},
			},
		},
		{
			dbUtxos: map[string]*account.UTXO{
				string(StandardUTXOKey(bc.Hash{V0: 3})): &account.UTXO{
					OutputID: bc.Hash{V0: 3},
				},
				string(ContractUTXOKey(bc.Hash{V0: 4})): &account.UTXO{
					OutputID: bc.Hash{V0: 4},
				},
			},
			unconfirmedUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 5},
					ControlProgram: []byte("smart contract"),
				},
				&account.UTXO{
					OutputID:       bc.Hash{V0: 6},
					ControlProgram: []byte{0x51},
				},
			},
			id:              "",
			unconfirmed:     true,
			isSmartContract: true,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 5},
					ControlProgram: []byte("smart contract"),
				},
				&account.UTXO{
					OutputID: bc.Hash{V0: 4},
				},
			},
		},
		{
			dbUtxos: map[string]*account.UTXO{
				string(StandardUTXOKey(bc.Hash{V0: 3})): &account.UTXO{
					OutputID: bc.Hash{V0: 3},
				},
				string(ContractUTXOKey(bc.Hash{V0: 4})): &account.UTXO{
					OutputID: bc.Hash{V0: 4},
				},
			},
			unconfirmedUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 5},
					ControlProgram: []byte("smart contract"),
				},
				&account.UTXO{
					OutputID:       bc.Hash{V0: 6},
					ControlProgram: []byte{0x51},
				},
			},
			id:              "",
			unconfirmed:     true,
			isSmartContract: false,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 6},
					ControlProgram: []byte{0x51},
				},
				&account.UTXO{
					OutputID: bc.Hash{V0: 3},
				},
			},
		},
	}

	w := &Wallet{Store: testStore}
	for i, c := range cases {
		for k, u := range c.dbUtxos {
			data, err := json.Marshal(u)
			if err != nil {
				t.Error(err)
			}
			testDB.Set([]byte(k), data)
		}

		acccountStore := NewMockAccountStore(testDB)
		w.AccountMgr = account.NewManager(acccountStore, nil)
		w.AccountMgr.AddUnconfirmedUtxo(c.unconfirmedUtxos)
		gotUtxos := w.GetAccountUtxos("", c.id, c.unconfirmed, c.isSmartContract, false)
		if !testutil.DeepEqual(gotUtxos, c.wantUtxos) {
			t.Errorf("case %d: got %v want %v", i, gotUtxos, c.wantUtxos)
		}

		for k := range c.dbUtxos {
			testDB.Delete([]byte(k))
		}
	}
}

func TestFilterAccountUtxo(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	testStore := NewMockWalletStore(testDB)
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	cases := []struct {
		dbPrograms map[string]*account.CtrlProgram
		input      []*account.UTXO
		wantUtxos  []*account.UTXO
	}{
		{
			dbPrograms: map[string]*account.CtrlProgram{},
			input:      []*account.UTXO{},
			wantUtxos:  []*account.UTXO{},
		},
		{
			dbPrograms: map[string]*account.CtrlProgram{
				"41533a013a2a37a64a4e15a772ab43bf3f5956d0d1f353946496788e7f40d0ff1796286a6f": &account.CtrlProgram{
					AccountID: "testAccount",
					Address:   "testAddress",
					KeyIndex:  53,
					Change:    true,
				},
			},
			input: []*account.UTXO{
				&account.UTXO{
					ControlProgram: []byte{0x00, 0x14, 0x62, 0x50, 0x18, 0xb6, 0x85, 0x77, 0xba, 0x9b, 0x26, 0x19, 0xc8, 0x1d, 0x2e, 0x96, 0xba, 0x22, 0xbe, 0x77, 0x77, 0xd7},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         3,
				},
				&account.UTXO{
					ControlProgram: []byte{0x91},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         4,
				},
			},
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					ControlProgram:      []byte{0x00, 0x14, 0x62, 0x50, 0x18, 0xb6, 0x85, 0x77, 0xba, 0x9b, 0x26, 0x19, 0xc8, 0x1d, 0x2e, 0x96, 0xba, 0x22, 0xbe, 0x77, 0x77, 0xd7},
					AssetID:             bc.AssetID{V0: 1},
					Amount:              3,
					AccountID:           "testAccount",
					Address:             "testAddress",
					ControlProgramIndex: 53,
					Change:              true,
				},
				&account.UTXO{
					ControlProgram: []byte{0x91},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         4,
				},
			},
		},
		{
			dbPrograms: map[string]*account.CtrlProgram{},
			input: []*account.UTXO{
				&account.UTXO{
					ControlProgram: []byte{0x00, 0x14, 0x62, 0x50, 0x18, 0xb6, 0x85, 0x77, 0xba, 0x9b, 0x26, 0x19, 0xc8, 0x1d, 0x2e, 0x96, 0xba, 0x22, 0xbe, 0x77, 0x77, 0xd7},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         3,
				},
				&account.UTXO{
					ControlProgram: []byte{0x91},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         3,
				},
			},
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					ControlProgram: []byte{0x91},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         3,
				},
			},
		},
		{
			dbPrograms: map[string]*account.CtrlProgram{
				"41533a013a2a37a64a4e15a772ab43bf3f5956d0d1f353946496788e7f40d0ff1796286a6f": &account.CtrlProgram{
					AccountID: "testAccount",
					Address:   "testAddress",
					KeyIndex:  53,
					Change:    true,
				},
				"41533a013adb4d86262c12ba70d50b3ca3ae102d5682436243bd1e8c79569603f75675036a": &account.CtrlProgram{
					AccountID: "testAccount2",
					Address:   "testAddress2",
					KeyIndex:  72,
					Change:    false,
				},
			},
			input: []*account.UTXO{
				&account.UTXO{
					ControlProgram: []byte{0x00, 0x14, 0x62, 0x50, 0x18, 0xb6, 0x85, 0x77, 0xba, 0x9b, 0x26, 0x19, 0xc8, 0x1d, 0x2e, 0x96, 0xba, 0x22, 0xbe, 0x77, 0x77, 0xd7},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         3,
				},
				&account.UTXO{
					ControlProgram: []byte{0x00, 0x14, 0x62, 0x50, 0x18, 0xb6, 0x85, 0x77, 0xba, 0x9b, 0x26, 0x19, 0xc8, 0x1d, 0x2e, 0x96, 0xba, 0x22, 0xbe, 0x77, 0x77, 0xd7},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         5,
				},
				&account.UTXO{
					ControlProgram: []byte{0x00, 0x14, 0xc6, 0xbf, 0x22, 0x19, 0x64, 0x2a, 0xc5, 0x9e, 0x5b, 0xe4, 0xeb, 0xdf, 0x5b, 0x22, 0x49, 0x56, 0xa7, 0x98, 0xa4, 0xdf},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         7,
				},
			},
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					ControlProgram:      []byte{0x00, 0x14, 0x62, 0x50, 0x18, 0xb6, 0x85, 0x77, 0xba, 0x9b, 0x26, 0x19, 0xc8, 0x1d, 0x2e, 0x96, 0xba, 0x22, 0xbe, 0x77, 0x77, 0xd7},
					AssetID:             bc.AssetID{V0: 1},
					Amount:              3,
					AccountID:           "testAccount",
					Address:             "testAddress",
					ControlProgramIndex: 53,
					Change:              true,
				},
				&account.UTXO{
					ControlProgram:      []byte{0x00, 0x14, 0x62, 0x50, 0x18, 0xb6, 0x85, 0x77, 0xba, 0x9b, 0x26, 0x19, 0xc8, 0x1d, 0x2e, 0x96, 0xba, 0x22, 0xbe, 0x77, 0x77, 0xd7},
					AssetID:             bc.AssetID{V0: 1},
					Amount:              5,
					AccountID:           "testAccount",
					Address:             "testAddress",
					ControlProgramIndex: 53,
					Change:              true,
				},
				&account.UTXO{
					ControlProgram:      []byte{0x00, 0x14, 0xc6, 0xbf, 0x22, 0x19, 0x64, 0x2a, 0xc5, 0x9e, 0x5b, 0xe4, 0xeb, 0xdf, 0x5b, 0x22, 0x49, 0x56, 0xa7, 0x98, 0xa4, 0xdf},
					AssetID:             bc.AssetID{V0: 1},
					Amount:              7,
					AccountID:           "testAccount2",
					Address:             "testAddress2",
					ControlProgramIndex: 72,
					Change:              false,
				},
			},
		},
	}

	accountStore := NewMockAccountStore(testDB)
	accountManager := account.NewManager(accountStore, nil)
	w := &Wallet{
		Store:      testStore,
		AccountMgr: accountManager,
	}
	for i, c := range cases {
		for s, p := range c.dbPrograms {
			data, err := json.Marshal(p)
			if err != nil {
				t.Error(err)
			}
			key, err := hex.DecodeString(s)
			if err != nil {
				t.Error(err)
			}
			testDB.Set(key, data)
		}

		gotUtxos := w.filterAccountUtxo(c.input)
		sort.Slice(gotUtxos[:], func(i, j int) bool {
			return gotUtxos[i].Amount < gotUtxos[j].Amount
		})

		if !testutil.DeepEqual(gotUtxos, c.wantUtxos) {
			t.Errorf("case %d: got %v want %v", i, gotUtxos, c.wantUtxos)
		}
		for s := range c.dbPrograms {
			key, err := hex.DecodeString(s)
			if err != nil {
				t.Error(err)
			}
			testDB.Delete(key)
		}
	}
}

func TestTxInToUtxos(t *testing.T) {
	cases := []struct {
		tx         *types.Tx
		statusFail bool
		wantUtxos  []*account.UTXO
	}{
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewCoinbaseInput([]byte{0x51}),
				},
				Outputs: []*types.TxOutput{
					types.NewIntraChainOutput(*consensus.BTMAssetID, 41250000000, []byte{0x51}),
				},
			}),
			statusFail: false,
			wantUtxos:  []*account.UTXO{},
		},
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 1}, bc.AssetID{V0: 1}, 1, 1, []byte{0x51}),
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 2}, bc.AssetID{V0: 1}, 3, 2, []byte{0x52}),
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 3}, *consensus.BTMAssetID, 5, 3, []byte{0x53}),
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 4}, *consensus.BTMAssetID, 7, 4, []byte{0x54}),
				},
				Outputs: []*types.TxOutput{
					types.NewIntraChainOutput(bc.AssetID{V0: 1}, 4, []byte{0x51}),
					types.NewIntraChainOutput(*consensus.BTMAssetID, 12, []byte{0x53}),
				},
			}),
			statusFail: false,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0x62, 0xf2, 0xc4, 0xa0, 0x9b, 0x47, 0xd1, 0x53, 0x58, 0xe7, 0x8c, 0x49, 0x36, 0x75, 0x02, 0xc1, 0x63, 0x46, 0x51, 0xc4, 0x0f, 0xef, 0x63, 0xe2, 0x7d, 0xe4, 0x3c, 0xb3, 0x2c, 0xfe, 0x97, 0xa2}),
					AssetID:        bc.AssetID{V0: 1},
					Amount:         1,
					ControlProgram: []byte{0x51},
					SourceID:       bc.Hash{V0: 1},
					SourcePos:      1,
				},
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0x99, 0x30, 0x35, 0x15, 0x9b, 0x0b, 0xcc, 0xdf, 0xbd, 0x15, 0x49, 0xb5, 0x2b, 0x4c, 0xc8, 0x71, 0x20, 0xe7, 0x2f, 0x77, 0x87, 0xcd, 0x88, 0x92, 0xba, 0xd8, 0x97, 0xfa, 0x4a, 0x2a, 0x1a, 0x10}),
					AssetID:        bc.AssetID{V0: 1},
					Amount:         3,
					ControlProgram: []byte{0x52},
					SourceID:       bc.Hash{V0: 2},
					SourcePos:      2,
				},
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0xe5, 0x21, 0x0a, 0x9f, 0x17, 0xa2, 0x3a, 0xcf, 0x47, 0x57, 0xf2, 0x16, 0x12, 0x9d, 0xd8, 0xea, 0x7a, 0x9f, 0x5a, 0x14, 0xa8, 0xd6, 0x32, 0x6f, 0xe8, 0xa8, 0x8e, 0xb7, 0xf4, 0xb4, 0xfb, 0xbd}),
					AssetID:        *consensus.BTMAssetID,
					Amount:         5,
					ControlProgram: []byte{0x53},
					SourceID:       bc.Hash{V0: 3},
					SourcePos:      3,
				},
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0x57, 0x65, 0x8d, 0x41, 0xed, 0xb7, 0x49, 0xd5, 0x1c, 0xf5, 0x95, 0x93, 0x16, 0x57, 0xf8, 0x66, 0x54, 0x1b, 0xb3, 0x45, 0x84, 0x19, 0x73, 0x2f, 0xb3, 0x3e, 0x44, 0x7c, 0x97, 0x33, 0x77, 0x12}),
					AssetID:        *consensus.BTMAssetID,
					Amount:         7,
					ControlProgram: []byte{0x54},
					SourceID:       bc.Hash{V0: 4},
					SourcePos:      4,
				},
			},
		},
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 1}, bc.AssetID{V0: 1}, 1, 1, []byte{0x51}),
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 2}, bc.AssetID{V0: 1}, 3, 2, []byte{0x52}),
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 3}, *consensus.BTMAssetID, 5, 3, []byte{0x53}),
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 4}, *consensus.BTMAssetID, 7, 4, []byte{0x54}),
				},
				Outputs: []*types.TxOutput{
					types.NewIntraChainOutput(bc.AssetID{V0: 1}, 4, []byte{0x51}),
					types.NewIntraChainOutput(*consensus.BTMAssetID, 12, []byte{0x53}),
				},
			}),
			statusFail: true,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0xe5, 0x21, 0x0a, 0x9f, 0x17, 0xa2, 0x3a, 0xcf, 0x47, 0x57, 0xf2, 0x16, 0x12, 0x9d, 0xd8, 0xea, 0x7a, 0x9f, 0x5a, 0x14, 0xa8, 0xd6, 0x32, 0x6f, 0xe8, 0xa8, 0x8e, 0xb7, 0xf4, 0xb4, 0xfb, 0xbd}),
					AssetID:        *consensus.BTMAssetID,
					Amount:         5,
					ControlProgram: []byte{0x53},
					SourceID:       bc.Hash{V0: 3},
					SourcePos:      3,
				},
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0x57, 0x65, 0x8d, 0x41, 0xed, 0xb7, 0x49, 0xd5, 0x1c, 0xf5, 0x95, 0x93, 0x16, 0x57, 0xf8, 0x66, 0x54, 0x1b, 0xb3, 0x45, 0x84, 0x19, 0x73, 0x2f, 0xb3, 0x3e, 0x44, 0x7c, 0x97, 0x33, 0x77, 0x12}),
					AssetID:        *consensus.BTMAssetID,
					Amount:         7,
					ControlProgram: []byte{0x54},
					SourceID:       bc.Hash{V0: 4},
					SourcePos:      4,
				},
			},
		},
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewVetoInput([][]byte{}, bc.Hash{V0: 1}, bc.AssetID{V0: 1}, 1, 1, []byte{0x51}, []byte("af594006a40837d9f028daabb6d589df0b9138daefad5683e5233c2646279217294a8d532e60863bcf196625a35fb8ceeffa3c09610eb92dcfb655a947f13269")),
				},
				Outputs: []*types.TxOutput{
					types.NewIntraChainOutput(bc.AssetID{V0: 1}, 1, []byte{0x51}),
				},
			}),
			statusFail: false,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.NewHash([32]byte{0x7c, 0x75, 0x7f, 0x03, 0x67, 0x9b, 0xc2, 0x8f, 0x8f, 0xbd, 0x04, 0x25, 0x72, 0x42, 0x4b, 0x0b, 0x2a, 0xa4, 0x0e, 0x10, 0x0a, 0x6e, 0x99, 0x0e, 0x6d, 0x58, 0x92, 0x1d, 0xdd, 0xbe, 0xeb, 0x1a}),
					AssetID:        bc.AssetID{V0: 1},
					Amount:         1,
					ControlProgram: []byte{0x51},
					Vote:           []byte("af594006a40837d9f028daabb6d589df0b9138daefad5683e5233c2646279217294a8d532e60863bcf196625a35fb8ceeffa3c09610eb92dcfb655a947f13269"),
					SourceID:       bc.Hash{V0: 1},
					SourcePos:      1,
				},
			},
		},
	}

	for i, c := range cases {
		if gotUtxos := txInToUtxos(c.tx, c.statusFail); !testutil.DeepEqual(gotUtxos, c.wantUtxos) {
			for k, v := range gotUtxos {
				data, _ := json.Marshal(v)
				fmt.Println(k, string(data))
			}
			for k, v := range c.wantUtxos {
				data, _ := json.Marshal(v)
				fmt.Println(k, string(data))
			}
			t.Errorf("case %d: got %v want %v", i, gotUtxos, c.wantUtxos)
		}
	}
}

func TestTxOutToUtxos(t *testing.T) {
	cases := []struct {
		tx          *types.Tx
		statusFail  bool
		blockHeight uint64
		wantUtxos   []*account.UTXO
	}{
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewCoinbaseInput([]byte{0x51}),
				},
				Outputs: []*types.TxOutput{
					types.NewIntraChainOutput(*consensus.BTMAssetID, 41250000000, []byte{0x51}),
				},
			}),
			statusFail:  false,
			blockHeight: 98,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 1728735075694344097, V1: 884766857607786922, V2: 12293210594955921685, V3: 11109045974561998790},
					AssetID:        *consensus.BTMAssetID,
					Amount:         41250000000,
					ControlProgram: []byte{0x51},
					SourceID:       bc.NewHash([32]byte{0xb4, 0x7e, 0x94, 0x31, 0x88, 0xfe, 0xd3, 0xe9, 0xac, 0x99, 0x7c, 0xfc, 0x99, 0x6d, 0xd7, 0x4d, 0x04, 0x10, 0x77, 0xcb, 0x1c, 0xf8, 0x95, 0x14, 0x00, 0xe3, 0x42, 0x00, 0x8d, 0x05, 0xec, 0xdc}),
					SourcePos:      0,
					ValidHeight:    consensus.MainNetParams.CoinbasePendingBlockNumber + 98,
				},
			},
		},
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 1}, bc.AssetID{V0: 1}, 5, 1, []byte{0x51}),
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 2}, *consensus.BTMAssetID, 7, 1, []byte{0x51}),
				},
				Outputs: []*types.TxOutput{
					types.NewIntraChainOutput(bc.AssetID{V0: 1}, 2, []byte{0x51}),
					types.NewIntraChainOutput(bc.AssetID{V0: 1}, 3, []byte{0x52}),
					types.NewIntraChainOutput(*consensus.BTMAssetID, 2, []byte{0x53}),
					types.NewIntraChainOutput(*consensus.BTMAssetID, 5, []byte{0x54}),
				},
			}),
			statusFail:  false,
			blockHeight: 0,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 8675398163687045889, V1: 7549510466747714094, V2: 13693077838209211470, V3: 6878568403630757599},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         2,
					ControlProgram: []byte{0x51},
					SourceID:       bc.Hash{V0: 968805671293010031, V1: 9297014342000792994, V2: 16963674611624423333, V3: 2728293460397542670},
					SourcePos:      0,
				},
				&account.UTXO{
					OutputID:       bc.Hash{V0: 10393356437681643401, V1: 233963481123580514, V2: 17312171816916184445, V3: 16199332547392196559},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         3,
					ControlProgram: []byte{0x52},
					SourceID:       bc.Hash{V0: 968805671293010031, V1: 9297014342000792994, V2: 16963674611624423333, V3: 2728293460397542670},
					SourcePos:      1,
				},
				&account.UTXO{
					OutputID:       bc.Hash{V0: 7067560744282869147, V1: 8991714784298240423, V2: 2595857933262917893, V3: 11490631006811252506},
					AssetID:        *consensus.BTMAssetID,
					Amount:         2,
					ControlProgram: []byte{0x53},
					SourceID:       bc.Hash{V0: 968805671293010031, V1: 9297014342000792994, V2: 16963674611624423333, V3: 2728293460397542670},
					SourcePos:      2,
				},
				&account.UTXO{
					OutputID:       bc.Hash{V0: 15425148469684856658, V1: 11568657474526458285, V2: 11930588814405533063, V3: 5058456773104068022},
					AssetID:        *consensus.BTMAssetID,
					Amount:         5,
					ControlProgram: []byte{0x54},
					SourceID:       bc.Hash{V0: 968805671293010031, V1: 9297014342000792994, V2: 16963674611624423333, V3: 2728293460397542670},
					SourcePos:      3,
				},
			},
		},
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 1}, bc.AssetID{V0: 1}, 5, 1, []byte{0x51}),
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 2}, *consensus.BTMAssetID, 7, 1, []byte{0x51}),
				},
				Outputs: []*types.TxOutput{
					types.NewIntraChainOutput(bc.AssetID{V0: 1}, 2, []byte{0x51}),
					types.NewIntraChainOutput(bc.AssetID{V0: 1}, 3, []byte{0x52}),
					types.NewIntraChainOutput(*consensus.BTMAssetID, 2, []byte{0x53}),
					types.NewIntraChainOutput(*consensus.BTMAssetID, 5, []byte{0x54}),
				},
			}),
			statusFail:  true,
			blockHeight: 0,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 7067560744282869147, V1: 8991714784298240423, V2: 2595857933262917893, V3: 11490631006811252506},
					AssetID:        *consensus.BTMAssetID,
					Amount:         2,
					ControlProgram: []byte{0x53},
					SourceID:       bc.Hash{V0: 968805671293010031, V1: 9297014342000792994, V2: 16963674611624423333, V3: 2728293460397542670},
					SourcePos:      2,
				},
				&account.UTXO{
					OutputID:       bc.Hash{V0: 15425148469684856658, V1: 11568657474526458285, V2: 11930588814405533063, V3: 5058456773104068022},
					AssetID:        *consensus.BTMAssetID,
					Amount:         5,
					ControlProgram: []byte{0x54},
					SourceID:       bc.Hash{V0: 968805671293010031, V1: 9297014342000792994, V2: 16963674611624423333, V3: 2728293460397542670},
					SourcePos:      3,
				},
			},
		},
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 1}, bc.AssetID{V0: 1}, 5, 1, []byte{0x51}),
					types.NewSpendInput([][]byte{}, bc.Hash{V0: 2}, *consensus.BTMAssetID, 7, 1, []byte{0x51}),
				},
				Outputs: []*types.TxOutput{
					types.NewIntraChainOutput(bc.AssetID{V0: 1}, 2, []byte{0x51}),
					types.NewCrossChainOutput(bc.AssetID{V0: 1}, 3, []byte{0x52}),
					types.NewIntraChainOutput(*consensus.BTMAssetID, 2, []byte{0x53}),
					types.NewCrossChainOutput(*consensus.BTMAssetID, 5, []byte{0x54}),
				},
			}),
			statusFail:  false,
			blockHeight: 0,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 8675398163687045889, V1: 7549510466747714094, V2: 13693077838209211470, V3: 6878568403630757599},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         2,
					ControlProgram: []byte{0x51},
					SourceID:       bc.Hash{V0: 968805671293010031, V1: 9297014342000792994, V2: 16963674611624423333, V3: 2728293460397542670},
					SourcePos:      0,
				},
				&account.UTXO{
					OutputID:       bc.Hash{V0: 7067560744282869147, V1: 8991714784298240423, V2: 2595857933262917893, V3: 11490631006811252506},
					AssetID:        *consensus.BTMAssetID,
					Amount:         2,
					ControlProgram: []byte{0x53},
					SourceID:       bc.Hash{V0: 968805671293010031, V1: 9297014342000792994, V2: 16963674611624423333, V3: 2728293460397542670},
					SourcePos:      2,
				},
			},
		},
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewCrossChainInput([][]byte{}, bc.Hash{V0: 1}, bc.AssetID{V0: 1}, 5, 1, 1, []byte("asset1"), []byte("IssuanceProgram")),
					types.NewCrossChainInput([][]byte{}, bc.Hash{V0: 2}, *consensus.BTMAssetID, 7, 1, 1, []byte("assetbtm"), []byte("IssuanceProgram"))},
				Outputs: []*types.TxOutput{
					types.NewIntraChainOutput(bc.AssetID{V0: 1}, 2, []byte{0x51}),
					types.NewIntraChainOutput(bc.AssetID{V0: 1}, 3, []byte{0x52}),
					types.NewIntraChainOutput(*consensus.BTMAssetID, 2, []byte{0x53}),
					types.NewIntraChainOutput(*consensus.BTMAssetID, 5, []byte{0x54}),
				},
			}),
			statusFail:  false,
			blockHeight: 0,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{15099088327605875240, 9219883424533839002, 14610773420520931246, 14899393216621986426},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         2,
					ControlProgram: []byte{0x51},
					SourceID:       bc.Hash{16280523637332892554, 3627898494554775182, 16212395834831293013, 3511838375364469081},
					SourcePos:      0,
				},
				&account.UTXO{
					OutputID:       bc.Hash{3610727630628260133, 13088239834060115701, 14968571476177322101, 7529789620153710893},
					AssetID:        bc.AssetID{V0: 1},
					Amount:         3,
					ControlProgram: []byte{0x52},
					SourceID:       bc.Hash{16280523637332892554, 3627898494554775182, 16212395834831293013, 3511838375364469081},
					SourcePos:      1,
				},
				&account.UTXO{
					OutputID:       bc.Hash{2034718018519539988, 16893043149780417913, 11926903829554245570, 3446441680088007327},
					AssetID:        *consensus.BTMAssetID,
					Amount:         2,
					ControlProgram: []byte{0x53},
					SourceID:       bc.Hash{16280523637332892554, 3627898494554775182, 16212395834831293013, 3511838375364469081},
					SourcePos:      2,
				},
				&account.UTXO{
					OutputID:       bc.Hash{7296157888262317106, 5789265653020263821, 1170213393196090227, 7665081318694049454},
					AssetID:        *consensus.BTMAssetID,
					Amount:         5,
					ControlProgram: []byte{0x54},
					SourceID:       bc.Hash{16280523637332892554, 3627898494554775182, 16212395834831293013, 3511838375364469081},
					SourcePos:      3,
				},
			},
		},
		{
			tx: types.NewTx(types.TxData{
				Inputs: []*types.TxInput{
					types.NewCoinbaseInput([]byte{0x51}),
				},
				Outputs: []*types.TxOutput{
					types.NewIntraChainOutput(*consensus.BTMAssetID, 0, []byte{0x51}),
					types.NewIntraChainOutput(*consensus.BTMAssetID, 3, []byte{0x52}),
				},
			}),
			statusFail:  false,
			blockHeight: 0,
			wantUtxos: []*account.UTXO{
				&account.UTXO{
					OutputID:       bc.Hash{V0: 17248080803965266442, V1: 11280159100206427956, V2: 14296992668077839045, V3: 10053986081220066749},
					AssetID:        *consensus.BTMAssetID,
					Amount:         3,
					ControlProgram: []byte{0x52},
					SourceID:       bc.Hash{V0: 14680680172533616824, V1: 32429899179491316, V2: 15399988966960786775, V3: 17411722803888206567},
					SourcePos:      1,
					ValidHeight:    consensus.MainNetParams.CoinbasePendingBlockNumber,
				},
			},
		},
	}

	for i, c := range cases {
		if gotUtxos := txOutToUtxos(c.tx, c.statusFail, c.blockHeight); !testutil.DeepEqual(gotUtxos, c.wantUtxos) {
			t.Errorf("case %d: got %v want %v", i, gotUtxos, c.wantUtxos)

			for j, u := range gotUtxos {
				t.Errorf("case %d: gotUtxos[%d] %v", i, j, u)
			}

			for j, u := range c.wantUtxos {
				t.Errorf("case %d: c.wantUtxos[%d] %v", i, j, u)
			}
		}
	}
}
