package config

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/bytom/vapor/testutil"
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
