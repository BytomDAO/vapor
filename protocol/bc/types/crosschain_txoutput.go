package types

import (
	"github.com/vapor/protocol/bc"
)

// CrossChainOutput satisfies the TypedOutput interface and represents a cross-chain transaction.
type CrossChainOutput struct {
	OutputCommitment
	// Unconsumed suffixes of the commitment and witness extensible strings.
	CommitmentSuffix []byte
}

// CrossChainOutput create a new output struct
func NewCrossChainOutput(assetID bc.AssetID, amount uint64, controlProgram []byte) *TxOutput {
	return &TxOutput{
		AssetVersion: currentAssetVersion,
		TypedOutput: &CrossChainOutput{
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

func (it *CrossChainOutput) OutputType() uint8 { return CrossChainOutputType }
