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
	MySQLConfig      common.MySQLConfig        `json:"mysql"`
	Chain            Chain                     `json:"chain"`
	VoteConf         *VoteRewardConfig         `json:"vote_reward"`
	OptionalNodeConf *OptionalNodeRewardConfig `json:"optional_node_reward"`
}

func DefaultConfig(isVoterReward bool) *Config {
	if isVoterReward {
		return &Config{
			VoteConf: DefaultVoteRewardConfig(),
		}
	}
	return &Config{
		OptionalNodeConf: DefaultOptionalNodeRewardConfig(),
	}
}

func ConfigFile() string {
	return path.Join("./", "reward.json")
}

type Chain struct {
	ChainID     string `json:"chain_id"`
	Name        string `json:"name"`
	Upstream    string `json:"upstream"`
	SyncSeconds uint64 `json:"sync_seconds"`
}

type VoteRewardConfig struct {
	XPub          string `json:"xpub"`
	Upstream      string `json:"upstream"`
	AccountID     string `json:"account_id"`
	Passwd        string `json:"password"`
	RewardRatio   int    `json:"reward_ratio"`
	MiningAddress string `json:"mining_adress"`
}

type OptionalNodeRewardConfig struct {
	TotalReward uint64 `json:"total_reward"`
}

func DefaultVoteRewardConfig() *VoteRewardConfig {
	return &VoteRewardConfig{
		Upstream: "http://127.0.0.1:9889",
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
