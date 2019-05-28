package config

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"

	"github.com/vapor/crypto/ed25519"
	"github.com/vapor/testutil"
)

func TestFederation(t *testing.T) {

	tmpDir, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatalf("failed to create temporary data folder: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := DefaultConfig()
	config.BaseConfig.RootDir = tmpDir

	if err := ExportFederationFile(config.FederationFile(), config); err != nil {
		t.Fatal(err)
	}

	loadConfig := &Config{
		Federation: &FederationConfig{},
	}

	if err := LoadFederationFile(config.FederationFile(), loadConfig); err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(config.Federation, loadConfig.Federation) {
		t.Fatalf("export: %v, load: %v", config.Federation, loadConfig.Federation)
	}
}

func TestFederationMulContract(t *testing.T) {
	wantContract := `
contract LockWithKeys(pubkey1,pubkey2,pubkey3,pubkey4,pubkey5,pubkey6: PublicKey) locks amount of asset {
	clause unlockWith2Sigs(sig1,sig2,sig3,sig4: Signature) {
		verify checkTxMultiSig([pubkey1,pubkey2,pubkey3,pubkey4,pubkey5,pubkey6],[sig1,sig2,sig3,sig4])
		unlock amount of asset
	}
}
`
	want := "206798460919e8dc7095ee8b9f9d65033ef3da8c2334813149da5a1e52e9c6da0720d72fb92fa13bf3e0deb39de3a47c8d6eef5584719f7877c82a4c009f78fddf9220983705ae71949c1a5d0fcf953658dd9ecc549f02c63e197b4d087ae31148097e20b58170b51ca61604028ba1cb412377dfc2bc6567c0afc84c83aae1c0c297d02220585e20143db413e45fbc82f03cb61f177e9916ef1df0012daa8cbf6dbb1025ce207f23aae65ee4307c38d342699e328f21834488e18191ebd66823d220b5a58303741b567a577a587a597a546bae5a7a5a7a5a7a5a7a5a7a5a7a566c7cad00c0"
	config := DefaultConfig()
	pubKeys := []ed25519.PublicKey{}
	for _, xpub := range config.Federation.Xpubs {
		pubKeys = append(pubKeys, xpub.PublicKey())
	}

	fedContract, err := generateFederationContract(pubKeys, config.Federation.Quorum)
	if err != nil {
		t.Fatal(err)
	}

	if wantContract != fedContract {
		t.Fatalf("fedContract: %s, wantContract:%s", fedContract, wantContract)
	}

	prog, err := GetFederationContractPrograms(pubKeys, config.Federation.Quorum)
	if err != nil {
		t.Fatal(err)
	}
	got := hex.EncodeToString(prog)
	if got != want {
		t.Fatalf("got: %s, want:%s", got, want)
	}
}
