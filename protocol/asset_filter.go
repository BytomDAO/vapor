package protocol

import (
	"strings"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/protocol/bc/types"
)

// AssetFilter is struct for allow open federation asset cross chain
type AssetFilter struct {
	whitelist map[string]struct{}
}

// NewAssetFilter returns a assetFilter according a whitelist,
// which is a strings list cancated via comma
func NewAssetFilter(whitelist string) *AssetFilter {
	af := &AssetFilter{whitelist: make(map[string]struct{})}
	af.whitelist[consensus.BTMAssetID.String()] = struct{}{}
	for _, assetID := range strings.Split(whitelist, ",") {
		af.whitelist[strings.ToLower(assetID)] = struct{}{}
	}
	return af
}

// IsDust implements the DustFilterer interface.
// It filters a transaction as long as there is one asset neither BTM or in the whitelist
// No need to check the output assets types becauese they must have been cover in input assets types
func (af *AssetFilter) IsDust(tx *types.Tx) bool {
	for _, input := range tx.Inputs {
		if _, ok := input.TypedInput.(*types.CrossChainInput); !ok {
			continue
		}

		assetID := input.AssetID()
		if _, ok := af.whitelist[assetID.String()]; !ok {
			return true
		}
	}

	return false
}
