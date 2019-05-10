package types

import (
	"github.com/vapor/protocol/bc"
)

// IntraChainTxOutput satisfies the TypedOutput interface and represents a intra-chain transaction.
type IntraChainTxOutput struct {
	OutputCommitment
	// Unconsumed suffixes of the commitment and witness extensible strings.
	CommitmentSuffix []byte
}

// NewIntraChainTxOutput create a new output struct
func NewIntraChainTxOutput(assetID bc.AssetID, amount uint64, controlProgram []byte) *TxOutput {
	return &TxOutput{
		AssetVersion: 1,
		TypedOutput: &IntraChainTxOutput{
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

func (it *IntraChainTxOutput) OutputType() uint8 { return IntraChainOutputType }
