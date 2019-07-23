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

func ExportConfigFile(configFile string, config *Config) error {
	buf := new(bytes.Buffer)

	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return err
	}

	return ioutil.WriteFile(configFile, buf.Bytes(), 0644)
}

func LoadConfigFile(configFile string, config *Config) error {
	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(config)
}
