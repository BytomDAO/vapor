package config

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/vapor/crypto/ed25519/chainkd"
)

type FederationDaemon struct {
	GinGonic    GinGonic    `json:"gin-gonic"`
	MySQLConfig MySQLConfig `json:"mysql"`
	Warders     []Warder    `json:"warders"`
	Mainchain   Chain       `json:"mainchain"`
	Sidechain   Chain       `json:"sidechain"`
}

type GinGonic struct {
	ListeningPort uint64 `json:"listening_port"`
	IsReleaseMode bool   `json:"is_release_mode"`
}

type MySQLConfig struct {
	Connection MySQLConnection `json:"connection"`
	LogMode    bool            `json:"log_mode"`
}

type MySQLConnection struct {
	Host     string `json:"host"`
	Port     uint   `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	DbName   string `json:"database"`
}

type Warder struct {
	Position uint8        `json:"position"`
	XPub     chainkd.XPub `json:"xpub"`
	HostPort string       `json:"host_port"`
	IsLocal  bool         `json:"is_local"`
}

type Chain struct {
	Name     string   `json:"name"`
	Upstream Upstream `json:"upstream"`
}

type Upstream struct {
	RPC       string `json:"rpc"`
	WebSocket string `json:"web_socket"`
}

func ExportFederationFile(fedFile string, config *Config) error {
	buf := new(bytes.Buffer)

	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config.Federation); err != nil {
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

	return json.NewDecoder(file).Decode(config.Federation)
}
