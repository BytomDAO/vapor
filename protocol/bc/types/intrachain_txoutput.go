package types

import (
	"github.com/vapor/protocol/bc"
)

// IntraChainTxOutput satisfies the TypedOutput interface and represents a intra-chain transaction.
type IntraChainTxOutput struct {
	// TODO:
	OutputCommitment
	// Unconsumed suffixes of the commitment and witness extensible strings.
	// TODO:
	CommitmentSuffix []byte
}

// NewIntraChainTxOutput create a new output struct
func NewIntraChainTxOutput(assetID bc.AssetID, amount uint64, controlProgram []byte) *TxOutput {
	return &TxOutput{
		AssetVersion: 1,
		OutputCommitment: OutputCommitment{
			AssetAmount: bc.AssetAmount{
				AssetId: &assetID,
				Amount:  amount,
			},
			VMVersion:      1,
			ControlProgram: controlProgram,
		},
	}
}

func (it *IntraChainTxOutput) TypedOutput() uint8 { return IntraChainOutputType }
