package database

import (
	"os"
	"testing"

	acc "github.com/vapor/account"
	dbm "github.com/vapor/database/leveldb"
)

func TestDeleteAccount(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	cases := []struct {
		accounts      []*acc.Account
		deleteAccount *acc.Account
		want          []*acc.Account
	}{
		{
			accounts:      []*acc.Account{},
			deleteAccount: &acc.Account{},
			want:          []*acc.Account{},
		},
		{
			accounts: []*acc.Account{},
			deleteAccount: &acc.Account{
				ID:    "id-1",
				Alias: "alias-1",
			},
			want: []*acc.Account{},
		},
		{
			accounts: []*acc.Account{
				&acc.Account{
					ID:    "id-1",
					Alias: "alias-1",
				},
				&acc.Account{
					ID:    "id-2",
					Alias: "alias-2",
				},
			},
			deleteAccount: &acc.Account{},
			want: []*acc.Account{
				&acc.Account{
					ID:    "id-1",
					Alias: "alias-1",
				},
				&acc.Account{
					ID:    "id-2",
					Alias: "alias-2",
				},
			},
		},
		{
			accounts: []*acc.Account{
				&acc.Account{
					ID:    "id-1",
					Alias: "alias-1",
				},
				&acc.Account{
					ID:    "id-2",
					Alias: "alias-2",
				},
			},
			deleteAccount: &acc.Account{
				ID:    "id-3",
				Alias: "alias-3",
			},
			want: []*acc.Account{
				&acc.Account{
					ID:    "id-1",
					Alias: "alias-1",
				},
				&acc.Account{
					ID:    "id-2",
					Alias: "alias-2",
				},
			},
		},
		{
			accounts: []*acc.Account{
				&acc.Account{
					ID:    "id-1",
					Alias: "alias-1",
				},
				&acc.Account{
					ID:    "id-2",
					Alias: "alias-2",
				},
			},
			deleteAccount: &acc.Account{
				ID:    "id-1",
				Alias: "alias-1",
			},
			want: []*acc.Account{
				&acc.Account{
					ID:    "id-2",
					Alias: "alias-2",
				},
			},
		},
	}

	for i, c := range cases {
		accountStore := NewAccountStore(testDB)
		as := accountStore.InitBatch()
		// store mock accounts
		for _, a := range c.accounts {
			if err := as.SetAccount(a); err != nil {
				t.Fatal(err)
			}
		}

		// delete account
		if err := as.DeleteAccount(c.deleteAccount); err != nil {
			t.Fatal(err)
		}

		if err := as.CommitBatch(); err != nil {
			t.Fatal(err)
		}

		// get account by deleteAccount.ID, it should print ErrFindAccount
		if _, err := as.GetAccountByID(c.deleteAccount.ID); err != acc.ErrFindAccount {
			t.Fatal(err)
		}

		for _, a := range c.want {
			if _, err := as.GetAccountByID(a.ID); err == acc.ErrFindAccount {
				t.Errorf("case %v: cann't find account, err: %v", i, err)
			}

			if _, err := as.GetAccountByAlias(a.Alias); err == acc.ErrFindAccount {
				t.Errorf("case %v: cann't find account, err: %v", i, err)
			}
		}
	}
}

// func TestDeleteStandardUTXO(t *testing.T) {
// 	testDB := dbm.NewDB("testdb", "leveldb", "temp")
// 	defer func() {
// 		testDB.Close()
// 		os.RemoveAll("temp")
// 	}()

// 	cases := []struct {
// 		utxos      []*acc.UTXO
// 		deleteUTXO *acc.UTXO
// 		want       []*acc.UTXO
// 	}{
// 		{
// 			utxos:      []*acc.UTXO{},
// 			deleteUTXO: &acc.UTXO{},
// 			want:       []*acc.UTXO{},
// 		},
// 		{
// 			utxos: []*acc.UTXO{
// 				&acc.UTXO{
// 					OutputID: bc.NewHash([32]byte{0x3e, 0x94, 0x5d, 0x35, 0x70, 0x30, 0xd4, 0x3b, 0x3d, 0xe3, 0xdd, 0x80, 0x67, 0x29, 0x9a, 0x5e, 0x09, 0xf9, 0xfb, 0x2b, 0xad, 0x5f, 0x92, 0xc8, 0x69, 0xd1, 0x42, 0x39, 0x74, 0x9a, 0xd1, 0x1c}),
// 				},
// 			},
// 			deleteUTXO: &acc.UTXO{
// 				OutputID: bc.NewHash([32]byte{0x3e, 0x94, 0x5d, 0x35, 0x70, 0x30, 0xd4, 0x3b, 0x3d, 0xe3, 0xdd, 0x80, 0x67, 0x29, 0x9a, 0x5e, 0x09, 0xf9, 0xfb, 0x2b, 0xad, 0x5f, 0x92, 0xc8, 0x69, 0xd1, 0x42, 0x39, 0x74, 0x9a, 0xd1, 0x1c}),
// 			},
// 			want: []*acc.UTXO{},
// 		},
// 	}

// 	accountStore := NewAccountStore(testDB)

// 	for i, c := range cases {
// 		accountStore.
// 	}
// }
