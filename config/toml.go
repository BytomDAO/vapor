package config

import (
	"path"

	cmn "github.com/tendermint/tmlibs/common"
)

/****** these are for production settings ***********/
func EnsureRoot(rootDir string, network string) {
	cmn.EnsureDir(rootDir, 0700)
	cmn.EnsureDir(rootDir+"/data", 0700)

	configFilePath := path.Join(rootDir, "config.toml")

	// Write default config file if missing.
	if !cmn.FileExists(configFilePath) {
		cmn.MustWriteFile(configFilePath, []byte(selectNetwork(network)), 0644)
	}
}

var defaultConfigTmpl = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml
fast_sync = true
db_backend = "leveldb"
api_addr = "0.0.0.0:9889"
moniker = ""
`

var soloNetConfigTmpl = `chain_id = "solonet"
[p2p]
laddr = "tcp://0.0.0.0:46658"
seeds = ""
`

var vaporNetConfigTmpl = `chain_id = "vapor"
[p2p]
laddr = "tcp://0.0.0.0:56656"
seeds = "52.82.77.112:56656,52.82.113.219:56656,52.82.119.51:56656"
`

// Select network seeds to merge a new string.
func selectNetwork(network string) string {
	switch network {
	case "vapor":
		return defaultConfigTmpl + vaporNetConfigTmpl
	default:
		return defaultConfigTmpl + soloNetConfigTmpl
	}
}
