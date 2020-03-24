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
asset_whitelist = ""
`

var testNetConfigTmpl = `chain_id = "testnet"
[p2p]
laddr = "tcp://0.0.0.0:56657"
seeds = "52.82.22.99:56657,52.82.24.162:56657"
[cross_chain]
asset_whitelist = "a0be88520f6b537b1244c10abe0eb917681ce9cd0c5a8aeebd867bdc882e72ab,9090fa534ec05423663be7c78e9571d7a04d6d5f567ce2df71eee838f944ff61,19994d5c0e967b899d822147a32d62c3dad0f27b10a15906e34eff53fe809c28,128d768781522a3c8b1d8b14975f04d6b0b2261b00af4f8b5aca49b1018036a8,ef7c49f350073ed51e2da6a76db5a9f368b0550d216ed33bde30e735fd811ac1"
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
