package config

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/toolbar/common"
)

type Config struct {
	MySQLConfig common.MySQLConfig `json:"mysql"`
	Chain       Chain              `json:"chain"`
	XPubs       []chainkd.XPub     `json:"xpubs"`
}

type Chain struct {
	Name        string `json:"name"`
	Upstream    string `json:"upstream"`
	SyncSeconds uint64 `json:"sync_seconds"`
}

func ExportConfigFile(fedFile string, config *Config) error {
	buf := new(bytes.Buffer)

	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return err
	}

	return ioutil.WriteFile(fedFile, buf.Bytes(), 0644)
}

func LoadCOnfigFile(fedFile string, config *Config) error {
	file, err := os.Open(fedFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(config)
}
