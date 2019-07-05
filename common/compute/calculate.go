package compute

import (
	"github.com/vapor/consensus"
	"github.com/vapor/protocol/bc/types"
)

// CalculateTxFee calculate transaction fee
func CalculateTxFee(tx *types.Tx) uint64 {
	var totalInputBTM, totalOutputBTM uint64
	for _, input := range tx.Inputs {
		if input.InputType() == types.CoinbaseInputType {
			return 0
		}
		if input.AssetID() == *consensus.BTMAssetID {
			totalInputBTM += input.Amount()
		}
	}

	for _, output := range tx.Outputs {
		if *output.AssetAmount().AssetId == *consensus.BTMAssetID {
			totalOutputBTM += output.AssetAmount().Amount
		}
	}
	return totalInputBTM - totalOutputBTM
}
