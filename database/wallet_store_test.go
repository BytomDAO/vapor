package database

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/vapor/blockchain/pseudohsm"
	"github.com/vapor/crypto/ed25519/chainkd"
)

func TestAccountIndexKey(t *testing.T) {
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
	if !reflect.DeepEqual(accountIndexKey(xpubs1), accountIndexKey(xpubs2)) {
		t.Fatal("accountIndexKey test err")
	}

	if reflect.DeepEqual(xpubs1, xpubs2) {
		t.Fatal("accountIndexKey test err")
	}
}
