package protocol

import (
	"strings"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/protocol/bc/types"
)

type assetFilter struct {
	whitelist map[string]struct{}
}

func NewAssetFilter(whitelist string) *assetFilter {
	af := &assetFilter{whitelist: make(map[string]struct{})}
	af.whitelist[consensus.BTMAssetID.String()] = struct{}{}
	for _, asset := range strings.Split(whitelist, ",") {
		af.whitelist[asset] = struct{}{}
	}
	return af
}

func (af *assetFilter) IsDust(tx *types.Tx) bool {
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
