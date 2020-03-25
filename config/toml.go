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

var mainNetConfigTmpl = `chain_id = "mainnet"
[p2p]
laddr = "tcp://0.0.0.0:56656"
seeds = "47.103.79.68:56656,47.103.13.86:56656,47.102.193.119:56656,47.103.17.22:56656"
[cross_chain]
asset_whitelist = "184e1cc4ee4845023888810a79eed7a42c02c544cf2c61ceac05e176d575bd46,78de44ffa1bce37b757c9eae8925b5f199dc4621b412ef0f3f46168865284a93"
`

var testNetConfigTmpl = `chain_id = "testnet"
[p2p]
laddr = "tcp://0.0.0.0:56657"
seeds = "52.82.7.233:56657,52.82.109.252:56657,52.82.29.30:56657"
[cross_chain]
asset_whitelist = ""
`

var soloNetConfigTmpl = `chain_id = "solonet"
[p2p]
laddr = "tcp://0.0.0.0:56658"
seeds = ""
[cross_chain]
asset_whitelist = ""
`

// Select network seeds to merge a new string.
func selectNetwork(network string) string {
	switch network {
	case "mainnet":
		return defaultConfigTmpl + mainNetConfigTmpl
	case "testnet":
		return defaultConfigTmpl + testNetConfigTmpl
	default:
		return defaultConfigTmpl + soloNetConfigTmpl
	}
}
