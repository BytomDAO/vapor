package config

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/toolbar/common"
)

func NewConfig() *Config {
	if len(os.Args) <= 1 {
		log.Fatal("Please setup the config file path")
	}

	return NewConfigWithPath(os.Args[1])
}

func NewConfigWithPath(path string) *Config {
	configFile, err := os.Open(path)
	if err != nil {
		log.WithFields(log.Fields{"err": err, "file_path": os.Args[1]}).Fatal("fail to open config file")
	}
	defer configFile.Close()

	cfg := &Config{}
	if err := json.NewDecoder(configFile).Decode(cfg); err != nil {
		log.WithField("err", err).Fatal("fail to decode config file")
	}

	return cfg
}

type Config struct {
	MySQLConfig      common.MySQLConfig        `json:"mysql"`
	Chain            Chain                     `json:"chain"`
	VoteConf         []VoteRewardConfig        `json:"vote_reward"`
	OptionalNodeConf *OptionalNodeRewardConfig `json:"optional_node_reward"`
}

func DefaultConfig(isVoterReward bool) *Config {
	if isVoterReward {
		return &Config{
			VoteConf: DefaultVoteRewardConfig(),
		}
	} else {
		return &Config{
			OptionalNodeConf: DefaultOptionalNodeRewardConfig(),
		}
	}

}

type Chain struct {
	ChainID     string `json:"chain_id"`
	Name        string `json:"name"`
	Upstream    string `json:"upstream"`
	SyncSeconds uint64 `json:"sync_seconds"`
}

type VoteRewardConfig struct {
	XPub          string `json:"xpub"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	AccountID     string `json:"account_id"`
	Passwd        string `json:"password"`
	RewardRatio   int    `json:"reward_ratio"`
	MiningAddress string `json:"mining_adress"`
}

type OptionalNodeRewardConfig struct {
	TotalReward uint64 `json:"total_reward"`
}

func DefaultVoteRewardConfig() []VoteRewardConfig {
	return []VoteRewardConfig{
		VoteRewardConfig{
			Host: "127.0.0.1",
			Port: 9889,
		},
	}
}

func DefaultOptionalNodeRewardConfig() *OptionalNodeRewardConfig {
	return &OptionalNodeRewardConfig{
		TotalReward: 30,
	}
}

func ExportFederationFile(fedFile string, config *Config) error {
	buf := new(bytes.Buffer)

	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return err
	}

	return ioutil.WriteFile(fedFile, buf.Bytes(), 0644)
}

func LoadFederationFile(fedFile string, config *Config) error {
	file, err := os.Open(fedFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(config)
}
