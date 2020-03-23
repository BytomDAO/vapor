package test

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	acc "github.com/bytom/vapor/account"
	"github.com/bytom/vapor/blockchain/signers"
	"github.com/bytom/vapor/config"
	"github.com/bytom/vapor/crypto/ed25519/chainkd"
	"github.com/bytom/vapor/database"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/event"
	"github.com/bytom/vapor/protocol"
	"github.com/bytom/vapor/testutil"
)

func TestCreateAccountWithUppercase(t *testing.T) {
	m := mockAccountManager(t)
	alias := "UPPER"
	account, err := m.Manager.Create([]chainkd.XPub{testutil.TestXPub}, 1, alias, signers.BIP0044)

	if err != nil {
		t.Fatal(err)
	}

	if account.Alias != strings.ToLower(alias) {
		t.Fatal("created account alias should be lowercase")
	}
}

func TestCreateAccountWithSpaceTrimed(t *testing.T) {
	m := mockAccountManager(t)
	alias := " with space "
	account, err := m.Manager.Create([]chainkd.XPub{testutil.TestXPub}, 1, alias, signers.BIP0044)

	if err != nil {
		t.Fatal(err)
	}

	if account.Alias != strings.TrimSpace(alias) {
		t.Fatal("created account alias should be lowercase")
	}

	nilAccount, err := m.Manager.FindByAlias(alias)
	if nilAccount != nil {
		t.Fatal("expected nil")
	}

	target, err := m.Manager.FindByAlias(strings.ToLower(strings.TrimSpace(alias)))
	if target == nil {
		t.Fatal("expected Account, but got nil")
	}
}

func TestCreateAccount(t *testing.T) {
	m := mockAccountManager(t)
	account, err := m.Manager.Create([]chainkd.XPub{testutil.TestXPub}, 1, "test-alias", signers.BIP0044)
	if err != nil {
		t.Fatal(err)
	}

	found, err := m.Manager.FindByID(account.ID)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if !testutil.DeepEqual(account, found) {
		t.Errorf("expected account %v to be recorded as %v", account, found)
	}
}

func TestCreateAccountReusedAlias(t *testing.T) {
	m := mockAccountManager(t)
	m.createTestAccount(t, "test-alias", nil)

	_, err := m.Manager.Create([]chainkd.XPub{testutil.TestXPub}, 1, "test-alias", signers.BIP0044)
	if errors.Root(err) != acc.ErrDuplicateAlias {
		t.Errorf("expected %s when reusing an alias, got %v", acc.ErrDuplicateAlias, err)
	}
}

func TestUpdateAccountAlias(t *testing.T) {
	oldAlias := "test-alias"
	newAlias := "my-alias"

	m := mockAccountManager(t)
	account := m.createTestAccount(t, oldAlias, nil)
	err := m.Manager.UpdateAccountAlias("testID", newAlias)
	if err == nil {
		t.Errorf("expected error when using an invalid account id")
	}

	err = m.Manager.UpdateAccountAlias(account.ID, oldAlias)
	if errors.Root(err) != acc.ErrDuplicateAlias {
		t.Errorf("expected %s when using a duplicate alias, got %v", acc.ErrDuplicateAlias, err)
	}

	err = m.Manager.UpdateAccountAlias(account.ID, newAlias)
	if err != nil {
		t.Errorf("expected account %v alias should be update", account)
	}

	updatedAccount, err := m.Manager.FindByID(account.ID)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if updatedAccount.Alias != newAlias {
		t.Errorf("alias:\ngot:  %v\nwant: %v", updatedAccount.Alias, newAlias)
	}

	if _, err = m.Manager.FindByAlias(oldAlias); errors.Root(err) != acc.ErrFindAccount {
		t.Errorf("expected %s when using a old alias, got %v", acc.ErrFindAccount, err)
	}
}

func TestDeleteAccount(t *testing.T) {
	m := mockAccountManager(t)

	account1, err := m.Manager.Create([]chainkd.XPub{testutil.TestXPub}, 1, "test-alias1", signers.BIP0044)
	if err != nil {
		t.Fatal(err)
	}

	account2, err := m.Manager.Create([]chainkd.XPub{testutil.TestXPub}, 1, "test-alias2", signers.BIP0044)
	if err != nil {
		t.Fatal(err)
	}

	found, err := m.Manager.FindByID(account1.ID)
	if err != nil {
		t.Errorf("expected account %v should be deleted", found)
	}

	if err = m.Manager.DeleteAccount(account2.ID); err != nil {
		t.Fatal(err)
	}

	found, err = m.Manager.FindByID(account2.ID)
	if err != nil {
		t.Errorf("expected account %v should be deleted", found)
	}
}

func TestFindByID(t *testing.T) {
	m := mockAccountManager(t)
	account := m.createTestAccount(t, "", nil)

	found, err := m.Manager.FindByID(account.ID)
	if err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(account, found) {
		t.Errorf("expected found account to be %v, instead found %v", account, found)
	}
}

func TestFindByAlias(t *testing.T) {
	m := mockAccountManager(t)
	account := m.createTestAccount(t, "some-alias", nil)

	found, err := m.Manager.FindByAlias("some-alias")
	if err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(account, found) {
		t.Errorf("expected found account to be %v, instead found %v", account, found)
	}
}

type mockAccManager struct {
	Manager *acc.Manager
}

func mockAccountManager(t *testing.T) *mockAccManager {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "memdb", dirPath)
	dispatcher := event.NewDispatcher()
	store := database.NewStore(testDB)
	accountStore := database.NewAccountStore(testDB)
	txPool := protocol.NewTxPool(store, nil, dispatcher)
	config.CommonConfig = config.DefaultConfig()
	chain, err := protocol.NewChain(store, txPool, nil, dispatcher)
	if err != nil {
		t.Fatal(err)
	}

	return &mockAccManager{acc.NewManager(accountStore, chain)}
}

func (m *mockAccManager) createTestAccount(t testing.TB, alias string, tags map[string]interface{}) *acc.Account {
	account, err := m.Manager.Create([]chainkd.XPub{testutil.TestXPub}, 1, alias, signers.BIP0044)
	if err != nil {
		t.Fatal(err)
	}

	return account
}
