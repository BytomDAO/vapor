package config

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

	"github.com/vapor/toolbar/common"
)

type Config struct {
	MySQLConfig common.MySQLConfig `json:"mysql"`
	NodeIP      string             `json:"node_ip"`
	RewardConf  *RewardConfig      `json:"reward_config"`
}

func ConfigFile() string {
	return path.Join("./", "reward.json")
}

type RewardConfig struct {
	XPub          string `json:"xpub"`
	AccountID     string `json:"account_id"`
	Passwd        string `json:"password"`
	RewardRatio   int    `json:"reward_ratio"`
	MiningAddress string `json:"mining_adress"`
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