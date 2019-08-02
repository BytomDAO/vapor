package config

import (
	"encoding/json"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/vapor/crypto/ed25519/chainkd"

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
	MySQLConfig common.MySQLConfig `json:"mysql"`
	Policy      Policy             `json:"policy"`
	Nodes       []Node             `json:"bootstrap_nodes"`
	API         API                `json:"api"`
}

type Policy struct {
	LantencyMS uint64 `json:"lantency_ms"`
}

type Node struct {
	Alias    string       `json:"alias"`
	HostPort string       `json:"host_port"`
	PubKey   chainkd.XPub `json:"pubkey"`
}

type API struct {
	HostPort      bool   `json:"host_port"`
	AccessToken   string `json:"access_token"`
	IsReleaseMode bool   `json:"is_release_mode"`
}
