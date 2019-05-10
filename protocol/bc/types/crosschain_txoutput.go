package types

import (
	"github.com/vapor/protocol/bc"
)

// CrossChainTxOutput satisfies the TypedOutput interface and represents a intra-chain transaction.
type CrossChainTxOutput struct {
	// TODO:
	OutputCommitment
	// Unconsumed suffixes of the commitment and witness extensible strings.
	// TODO:
	CommitmentSuffix []byte
}

// CrossChainTxOutput create a new output struct
func NewCrossChainTxOutput(assetID bc.AssetID, amount uint64, controlProgram []byte) *TxOutput {
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

func (it *CrossChainTxOutput) TypedOutput() uint8 { return CrossChainOutputType }
