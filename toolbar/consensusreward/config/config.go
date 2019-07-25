package config

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
)

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

type Config struct {
	NodeIP     string        `json:"node_ip"`
	ChainID    string        `json:"chain_id"`
	RewardConf *RewardConfig `json:"reward_config"`
}

func Default() *Config {
	return &Config{
		RewardConf: &RewardConfig{
			Node: []NodeConfig{
				{
					XPub:    "",
					Address: "",
				},
			},
		},
	}
}

type RewardConfig struct {
	Node      []NodeConfig `json:"node"`
	AccountID string       `json:"account_id"`
	Password  string       `json:"password"`
}

type NodeConfig struct {
	XPub    string `json:"xpub"`
	Address string `json:"address"`
}
