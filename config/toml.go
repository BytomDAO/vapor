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
asset_whitelist = "47fcd4d7c22d1d38931a6cd7767156babbd5f05bbbb3f7d3900635b56eb1b67e,184e1cc4ee4845023888810a79eed7a42c02c544cf2c61ceac05e176d575bd46,78de44ffa1bce37b757c9eae8925b5f199dc4621b412ef0f3f46168865284a93,bda946b3110fa46fd94346ce3f05f0760f1b9de72e238835bc4d19f9d64f1742,25f2069140fa3ff4d6e0dc1d0fcaa11ace01eb721f115f0f1a5a3782db597fb1,c4644dd6643475d57ed624f63129ab815f282b61f4bb07646d73423a6e1a1563"
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
