package arithmetic

import (
	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/math/checked"
	"github.com/bytom/vapor/protocol/bc/types"
)

// CalculateTxFee calculate transaction fee
func CalculateTxFee(tx *types.Tx) (fee uint64, err error) {
	var ok bool
	for _, input := range tx.Inputs {
		if input.InputType() == types.CoinbaseInputType {
			return 0, nil
		}
		if input.AssetID() == *consensus.BTMAssetID {
			if fee, ok = checked.AddUint64(fee, input.Amount()); !ok {
				return 0, checked.ErrOverflow
			}
		}
	}

	for _, output := range tx.Outputs {
		if *output.AssetAmount().AssetId == *consensus.BTMAssetID {
			if fee, ok = checked.SubUint64(fee, output.AssetAmount().Amount); !ok {
				return 0, checked.ErrOverflow
			}
		}
	}
	return
}
