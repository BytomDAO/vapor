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

	accountStore := NewAccountStore(testDB)
	for i, c := range cases {
		// store mock accounts
		for _, a := range c.accounts {
			if err := accountStore.SetAccount(a); err != nil {
				t.Fatal(err)
			}
		}

		// delete account
		if err := accountStore.DeleteAccount(c.deleteAccount); err != nil {
			t.Fatal(err)
		}

		// get account by deleteAccount.ID, it should print ErrFindAccount
		if _, err := accountStore.GetAccountByID(c.deleteAccount.ID); err != acc.ErrFindAccount {
			t.Fatal(err)
		}

		for _, a := range c.want {
			if _, err := accountStore.GetAccountByID(a.ID); err == acc.ErrFindAccount {
				t.Errorf("case %v: cann't find account, err: %v", i, err)
			}

			if _, err := accountStore.GetAccountByAlias(a.Alias); err == acc.ErrFindAccount {
				t.Errorf("case %v: cann't find account, err: %v", i, err)
			}
		}
	}
}
