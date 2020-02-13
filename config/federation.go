package config

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
)

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

type FederationAssetFilter struct {
	whitelist map[string]struct{}
}

func NewFederationAssetFilter(whitelist []*bc.AssetID) *FederationAssetFilter {
	f := &FederationAssetFilter{whitelist: make(map[string]struct{})}
	f.whitelist[consensus.BTMAssetID.String()] = struct{}{}
	for _, asset := range whitelist {
		f.whitelist[asset.String()] = struct{}{}
	}
	return f
}

func (f *FederationAssetFilter) IsDust(tx *types.Tx) bool {
	for _, input := range tx.Inputs {
		assetID := input.AssetID()
		if _, ok := f.whitelist[assetID.String()]; !ok {
			return true
		}
	}

	for _, output := range tx.Outputs {
		assetID := output.AssetAmount().AssetId
		if _, ok := f.whitelist[assetID.String()]; !ok {
			return true
		}
	}

	return false
}
