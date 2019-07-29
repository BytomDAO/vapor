package config

import (
	"encoding/json"
	"os"
)

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
	RewardConf *RewardConfig `json:"reward_config"`
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
