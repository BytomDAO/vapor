package config

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/vapor/toolbar/common"
)

type Config struct {
	MySQLConfig common.MySQLConfig `json:"mysql"`
	NodeIP      string             `json:"node_ip"`
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

func LoadConfigFile(fedFile string, config *Config) error {
	file, err := os.Open(fedFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(config)
}
