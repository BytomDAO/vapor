package types

import (
	"github.com/vapor/protocol/bc"
)

// CrossChainTxOutput satisfies the TypedOutput interface and represents a cross-chain transaction.
type CrossChainTxOutput struct {
	OutputCommitment
	// Unconsumed suffixes of the commitment and witness extensible strings.
	CommitmentSuffix []byte
}

// CrossChainTxOutput create a new output struct
func NewCrossChainTxOutput(assetID bc.AssetID, amount uint64, controlProgram []byte) *TxOutput {
	return &TxOutput{
		AssetVersion: 1,
		TypedOutput: &CrossChainTxOutput{
			OutputCommitment: OutputCommitment{
				AssetAmount: bc.AssetAmount{
					AssetId: &assetID,
					Amount:  amount,
				},
				VMVersion:      1,
				ControlProgram: controlProgram,
			},
		},
	}
}

func (it *CrossChainTxOutput) OutputType() uint8 { return CrossChainOutputType }
