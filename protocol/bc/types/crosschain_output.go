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

// NewCrossChainOutput create a new output struct
func NewCrossChainOutput(assetID bc.AssetID, amount uint64, controlProgram []byte) *TxOutput {
	return &TxOutput{
		AssetVersion: 1,
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
