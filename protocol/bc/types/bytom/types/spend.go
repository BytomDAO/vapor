package types

import (
	"github.com/bytom/protocol/bc/types/bytom"
)

// SpendInput satisfies the TypedInput interface and represents a spend transaction.
type SpendInput struct {
	SpendCommitmentSuffix []byte   // The unconsumed suffix of the output commitment
	Arguments             [][]byte // Witness
	SpendCommitment
}

// NewSpendInput create a new SpendInput struct.
func NewSpendInput(arguments [][]byte, sourceID bytom.Hash, assetID bytom.AssetID, amount, sourcePos uint64, controlProgram []byte) *TxInput {
	sc := SpendCommitment{
		AssetAmount: bytom.AssetAmount{
			AssetId: &assetID,
			Amount:  amount,
		},
		SourceID:       sourceID,
		SourcePosition: sourcePos,
		VMVersion:      1,
		ControlProgram: controlProgram,
	}
	return &TxInput{
		AssetVersion: 1,
		TypedInput: &SpendInput{
			SpendCommitment: sc,
			Arguments:       arguments,
		},
	}
}

// InputType is the interface function for return the input type.
func (si *SpendInput) InputType() uint8 { return SpendInputType }
