package config

import (
	"encoding/json"
	"os"

	log "github.com/sirupsen/logrus"

	vaporJson "github.com/vapor/encoding/json"
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
	API            API                `json:"api"`
	MySQLConfig    common.MySQLConfig `json:"mysql"`
	FederationProg vaporJson.HexBytes `json:"federation_prog"`
	Mainchain      Chain              `json:"mainchain"`
	Sidechain      Chain              `json:"sidechain"`
}

type API struct {
	IsReleaseMode bool `json:"is_release_mode"`
}

type Chain struct {
	Name          string `json:"name"`
	Upstream      string `json:"upstream"`
	SyncSeconds   uint64 `json:"sync_seconds"`
	Confirmations uint64 `json:"confirmations"`
}
