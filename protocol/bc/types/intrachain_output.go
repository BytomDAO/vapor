package types

import (
	"github.com/bytom/vapor/protocol/bc"
)

// IntraChainOutput satisfies the TypedOutput interface and represents a intra-chain transaction.
type IntraChainOutput struct {
	OutputCommitment
	// Unconsumed suffixes of the commitment and witness extensible strings.
	CommitmentSuffix []byte
}

// NewIntraChainOutput create a new output struct
func NewIntraChainOutput(assetID bc.AssetID, amount uint64, controlProgram []byte) *TxOutput {
	return &TxOutput{
		AssetVersion: 1,
		TypedOutput: &IntraChainOutput{
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

// OutputType implement the txout interface
func (it *IntraChainOutput) OutputType() uint8 { return IntraChainOutputType }
