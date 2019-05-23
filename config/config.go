package config

import (
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/crypto/ed25519"
	"github.com/vapor/crypto/ed25519/chainkd"
)

var (
	// CommonConfig means config object
	CommonConfig *Config
)

type Config struct {
	// Top level options use an anonymous struct
	BaseConfig `mapstructure:",squash"`
	// Options for services
	P2P        *P2PConfig        `mapstructure:"p2p"`
	Wallet     *WalletConfig     `mapstructure:"wallet"`
	Auth       *RPCAuthConfig    `mapstructure:"auth"`
	Web        *WebConfig        `mapstructure:"web"`
	Websocket  *WebsocketConfig  `mapstructure:"ws"`
	Federation *FederationConfig `mapstructure:"federation"`
}

// Default configurable parameters.
func DefaultConfig() *Config {
	return &Config{
		BaseConfig: DefaultBaseConfig(),
		P2P:        DefaultP2PConfig(),
		Wallet:     DefaultWalletConfig(),
		Auth:       DefaultRPCAuthConfig(),
		Web:        DefaultWebConfig(),
		Websocket:  DefaultWebsocketConfig(),
		Federation: DefaultFederationConfig(),
	}
}

// Set the RootDir for all Config structs
func (cfg *Config) SetRoot(root string) *Config {
	cfg.BaseConfig.RootDir = root
	return cfg
}

// NodeKey retrieves the currently configured private key of the node, checking
// first any manually set key, falling back to the one found in the configured
// data folder. If no key can be found, a new one is generated.
func (cfg *Config) NodeKey() (string, error) {
	// Use any specifically configured key.
	if cfg.P2P.PrivateKey != "" {
		return cfg.P2P.PrivateKey, nil
	}

	keyFile := rootify(cfg.P2P.NodeKeyFile, cfg.BaseConfig.RootDir)
	buf := make([]byte, ed25519.PrivateKeySize*2)
	fd, err := os.Open(keyFile)
	defer fd.Close()
	if err == nil {
		if _, err = io.ReadFull(fd, buf); err == nil {
			return string(buf), nil
		}
	}

	log.WithField("err", err).Warning("key file access failed")
	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return "", err
	}

	if err = ioutil.WriteFile(keyFile, []byte(privKey.String()), 0600); err != nil {
		return "", err
	}
	return privKey.String(), nil
}

//-----------------------------------------------------------------------------
// BaseConfig
type BaseConfig struct {
	// The root directory for all data.
	// This should be set in viper so it can unmarshal into this struct
	RootDir string `mapstructure:"home"`

	//The ID of the network to json
	ChainID string `mapstructure:"chain_id"`

	//log level to set
	LogLevel string `mapstructure:"log_level"`

	// A custom human readable name for this node
	Moniker string `mapstructure:"moniker"`

	// TCP or UNIX socket address for the profiling server to listen on
	ProfListenAddress string `mapstructure:"prof_laddr"`

	Mining bool `mapstructure:"mining"`

	// Database backend: leveldb | memdb
	DBBackend string `mapstructure:"db_backend"`

	// Database directory
	DBPath string `mapstructure:"db_dir"`

	// Keystore directory
	KeysPath string `mapstructure:"keys_dir"`

	ApiAddress string `mapstructure:"api_addr"`

	VaultMode bool `mapstructure:"vault_mode"`

	// log file name
	LogFile string `mapstructure:"log_file"`

	// Cipher Service Provider
	// CipherServiceProvider string `mapstructure:"csp"`
}

// Default configurable base parameters.
func DefaultBaseConfig() BaseConfig {
	return BaseConfig{
		Moniker:           "anonymous",
		ProfListenAddress: "",
		Mining:            false,
		DBBackend:         "leveldb",
		DBPath:            "data",
		KeysPath:          "keystore",
	}
}

func (b BaseConfig) DBDir() string {
	return rootify(b.DBPath, b.RootDir)
}

func (b BaseConfig) KeysDir() string {
	return rootify(b.KeysPath, b.RootDir)
}

// P2PConfig
type P2PConfig struct {
	ListenAddress    string `mapstructure:"laddr"`
	Seeds            string `mapstructure:"seeds"`
	PrivateKey       string `mapstructure:"node_key"`
	NodeKeyFile      string `mapstructure:"node_key_file"`
	SkipUPNP         bool   `mapstructure:"skip_upnp"`
	LANDiscover      bool   `mapstructure:"lan_discoverable"`
	MaxNumPeers      int    `mapstructure:"max_num_peers"`
	HandshakeTimeout int    `mapstructure:"handshake_timeout"`
	DialTimeout      int    `mapstructure:"dial_timeout"`
	ProxyAddress     string `mapstructure:"proxy_address"`
	ProxyUsername    string `mapstructure:"proxy_username"`
	ProxyPassword    string `mapstructure:"proxy_password"`
	KeepDial         string `mapstructure:"keep_dial"`
}

// Default configurable p2p parameters.
func DefaultP2PConfig() *P2PConfig {
	return &P2PConfig{
		ListenAddress:    "tcp://0.0.0.0:46656",
		NodeKeyFile:      "nodekey",
		SkipUPNP:         false,
		LANDiscover:      true,
		MaxNumPeers:      50,
		HandshakeTimeout: 30,
		DialTimeout:      3,
		ProxyAddress:     "",
		ProxyUsername:    "",
		ProxyPassword:    "",
	}
}

//-----------------------------------------------------------------------------
type WalletConfig struct {
	Disable  bool   `mapstructure:"disable"`
	Rescan   bool   `mapstructure:"rescan"`
	TxIndex  bool   `mapstructure:"txindex"`
	MaxTxFee uint64 `mapstructure:"max_tx_fee"`
}

type RPCAuthConfig struct {
	Disable bool `mapstructure:"disable"`
}

type WebConfig struct {
	Closed bool `mapstructure:"closed"`
}

type WebsocketConfig struct {
	MaxNumWebsockets     int `mapstructure:"max_num_websockets"`
	MaxNumConcurrentReqs int `mapstructure:"max_num_concurrent_reqs"`
}

type FederationConfig struct {
	Xpubs  []chainkd.XPub `json:"xpubs"`
	Quorum int            `json:"quorum"`
}

// Default configurable rpc's auth parameters.
func DefaultRPCAuthConfig() *RPCAuthConfig {
	return &RPCAuthConfig{
		Disable: false,
	}
}

// Default configurable web parameters.
func DefaultWebConfig() *WebConfig {
	return &WebConfig{
		Closed: false,
	}
}

// Default configurable wallet parameters.
func DefaultWalletConfig() *WalletConfig {
	return &WalletConfig{
		Disable:  false,
		Rescan:   false,
		TxIndex:  false,
		MaxTxFee: uint64(1000000000),
	}
}

func DefaultWebsocketConfig() *WebsocketConfig {
	return &WebsocketConfig{
		MaxNumWebsockets:     25,
		MaxNumConcurrentReqs: 20,
	}
}

func DefaultFederationConfig() *FederationConfig {
	return &FederationConfig{
		Xpubs: []chainkd.XPub{
			xpub("7f23aae65ee4307c38d342699e328f21834488e18191ebd66823d220b5a58303496c9d09731784372bade78d5e9a4a6249b2cfe2e3a85464e5a4017aa5611e47"),
			xpub("585e20143db413e45fbc82f03cb61f177e9916ef1df0012daa8cbf6dbb1025ce8f98e51ae319327b63505b64fdbbf6d36ef916d79e6dd67d51b0bfe76fe544c5"),
			xpub("b58170b51ca61604028ba1cb412377dfc2bc6567c0afc84c83aae1c0c297d0227ccf568561df70851f4144bbf069b525129f2434133c145e35949375b22a6c9d"),
			xpub("983705ae71949c1a5d0fcf953658dd9ecc549f02c63e197b4d087ae31148097ece816bbc60d9012c316139fc550fa0f4b00beb0887f6b152f7a69bc8f392b9fa"),
			xpub("d72fb92fa13bf3e0deb39de3a47c8d6eef5584719f7877c82a4c009f78fddf924d9706d48f15b2c782ec80b6bdd621a1f7ba2a0044b0e6f92245de9436885cb9"),
			xpub("6798460919e8dc7095ee8b9f9d65033ef3da8c2334813149da5a1e52e9c6da07ba7d0e7379baaa0c8bdcb21890a54e6b7290bee077c645ee4b74b0c1ae9da59a"),
		},
		Quorum: 4,
	}
}

func xpub(str string) (xpub chainkd.XPub) {
	if err := xpub.UnmarshalText([]byte(str)); err != nil {
		log.Panicf("Fail converts a string to xpub")
	}
	return xpub
}

//-----------------------------------------------------------------------------
// Utils

// helper function to make config creation independent of root dir
func rootify(path, root string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

// DefaultDataDir is the default data directory to use for the databases and other
// persistence requirements.
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home == "" {
		return "./.vapor"
	}
	switch runtime.GOOS {
	case "darwin":
		// In order to be compatible with old data path,
		// copy the data from the old path to the new path
		oldPath := filepath.Join(home, "Library", "Vapor")
		newPath := filepath.Join(home, "Library", "Application Support", "Vapor")
		if !isFolderNotExists(oldPath) && isFolderNotExists(newPath) {
			if err := os.Rename(oldPath, newPath); err != nil {
				log.Errorf("DefaultDataDir: %v", err)
				return oldPath
			}
		}
		return newPath
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "Vapor")
	default:
		return filepath.Join(home, ".vapor")
	}
}

func isFolderNotExists(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
