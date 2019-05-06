package account

import (
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/vapor/blockchain/pseudohsm"
	"github.com/vapor/blockchain/signers"
	"github.com/vapor/common"
	"github.com/vapor/config"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/database"
	dbm "github.com/vapor/database/db"
	_ "github.com/vapor/database/leveldb"
	"github.com/vapor/errors"
	"github.com/vapor/protocol"
	"github.com/vapor/testutil"
)

func TestCreateAccountWithUppercase(t *testing.T) {
	m := mockAccountManager(t)
	alias := "UPPER"
	account, err := m.Create([]chainkd.XPub{testutil.TestXPub}, 1, alias, signers.BIP0044)

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
	account, err := m.Create([]chainkd.XPub{testutil.TestXPub}, 1, alias, signers.BIP0044)

	if err != nil {
		t.Fatal(err)
	}

	if account.Alias != strings.TrimSpace(alias) {
		t.Fatal("created account alias should be lowercase")
	}

	nilAccount, err := m.FindByAlias(alias)
	if nilAccount != nil {
		t.Fatal("expected nil")
	}

	target, err := m.FindByAlias(strings.ToLower(strings.TrimSpace(alias)))
	if target == nil {
		t.Fatal("expected Account, but got nil")
	}
}

func TestCreateAccount(t *testing.T) {
	m := mockAccountManager(t)
	account, err := m.Create([]chainkd.XPub{testutil.TestXPub}, 1, "test-alias", signers.BIP0044)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	found, err := m.FindByID(account.ID)
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

	_, err := m.Create([]chainkd.XPub{testutil.TestXPub}, 1, "test-alias", signers.BIP0044)
	if errors.Root(err) != ErrDuplicateAlias {
		t.Errorf("expected %s when reusing an alias, got %v", ErrDuplicateAlias, err)
	}
}

func TestUpdateAccountAlias(t *testing.T) {
	oldAlias := "test-alias"
	newAlias := "my-alias"

	m := mockAccountManager(t)
	account := m.createTestAccount(t, oldAlias, nil)
	if err := m.UpdateAccountAlias("testID", newAlias); err == nil {
		t.Fatal("expected error when using an invalid account id")
	}

	err := m.UpdateAccountAlias(account.ID, oldAlias)
	if errors.Root(err) != ErrDuplicateAlias {
		t.Errorf("expected %s when using a duplicate alias, got %v", ErrDuplicateAlias, err)
	}

	if err := m.UpdateAccountAlias(account.ID, newAlias); err != nil {
		t.Errorf("expected account %v alias should be update", account)
	}

	updatedAccount, err := m.FindByID(account.ID)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if updatedAccount.Alias != newAlias {
		t.Fatalf("alias:\ngot:  %v\nwant: %v", updatedAccount.Alias, newAlias)
	}

	if _, err = m.FindByAlias(oldAlias); errors.Root(err) != ErrFindAccount {
		t.Errorf("expected %s when using a old alias, got %v", ErrFindAccount, err)
	}
}

func TestDeleteAccount(t *testing.T) {
	m := mockAccountManager(t)

	account1, err := m.Create([]chainkd.XPub{testutil.TestXPub}, 1, "test-alias1", signers.BIP0044)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	account2, err := m.Create([]chainkd.XPub{testutil.TestXPub}, 1, "test-alias2", signers.BIP0044)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	found, err := m.FindByID(account1.ID)
	if err != nil {
		t.Errorf("expected account %v should be deleted", found)
	}

	if err = m.DeleteAccount(account2.ID); err != nil {
		testutil.FatalErr(t, err)
	}

	found, err = m.FindByID(account2.ID)
	if err != nil {
		t.Errorf("expected account %v should be deleted", found)
	}
}

func TestFindByID(t *testing.T) {
	m := mockAccountManager(t)
	account := m.createTestAccount(t, "", nil)

	found, err := m.FindByID(account.ID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !testutil.DeepEqual(account, found) {
		t.Errorf("expected found account to be %v, instead found %v", account, found)
	}
}

func TestFindByAlias(t *testing.T) {
	m := mockAccountManager(t)
	account := m.createTestAccount(t, "some-alias", nil)

	found, err := m.FindByAlias("some-alias")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !testutil.DeepEqual(account, found) {
		t.Errorf("expected found account to be %v, instead found %v", account, found)
	}
}

func TestGetAccountIndexKey(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "TestAccount")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	xpub1, _, err := hsm.XCreate("TestAccountIndex1", "password", "en")
	if err != nil {
		t.Fatal(err)
	}

	xpub2, _, err := hsm.XCreate("TestAccountIndex2", "password", "en")
	if err != nil {
		t.Fatal(err)
	}

	xpubs1 := []chainkd.XPub{xpub1.XPub, xpub2.XPub}
	xpubs2 := []chainkd.XPub{xpub2.XPub, xpub1.XPub}
	if !reflect.DeepEqual(GetAccountIndexKey(xpubs1), GetAccountIndexKey(xpubs2)) {
		t.Fatal("GetAccountIndexKey test err")
	}

	if reflect.DeepEqual(xpubs1, xpubs2) {
		t.Fatal("GetAccountIndexKey test err")
	}
}

func mockAccountManager(t *testing.T) *Manager {

	config.CommonConfig = config.DefaultConfig()
	consensus.SoloNetParams.Signer = "78673764e0ba91a4c5ba9ec0c8c23c69e3d73bf27970e05e0a977e81e13bde475264d3b177a96646bc0ce517ae7fd63504c183ab6d330dea184331a4cf5912d5"
	config.CommonConfig.Consensus.SelfVoteSigners = append(config.CommonConfig.Consensus.SelfVoteSigners, "vsm1qkm743xmgnvh84pmjchq2s4tnfpgu9ae2f9slep")
	config.CommonConfig.Consensus.XPrv = "a8e281b615809046698fb0b0f2804a36d824d48fa443350f10f1b80649d39e5f1e85cf9855548915e36137345910606cbc8e7dd8497c831dce899ee6ac112445"
	for _, v := range config.CommonConfig.Consensus.SelfVoteSigners {
		address, err := common.DecodeAddress(v, &consensus.SoloNetParams)
		if err != nil {
			t.Fatal(err)
		}
		config.CommonConfig.Consensus.Signers = append(config.CommonConfig.Consensus.Signers, address)
	}
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "memdb", "temp")
	defer os.RemoveAll("temp")

	store := database.NewStore(testDB)
	txPool := protocol.NewTxPool(store)
	chain, err := protocol.NewChain(store, txPool)
	if err != nil {
		t.Fatal(err)
	}

	return NewManager(testDB, chain)
}

func (m *Manager) createTestAccount(t testing.TB, alias string, tags map[string]interface{}) *Account {
	account, err := m.Create([]chainkd.XPub{testutil.TestXPub}, 1, alias, signers.BIP0044)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return account

}
