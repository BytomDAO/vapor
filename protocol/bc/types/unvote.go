package types

import (
	"github.com/vapor/protocol/bc"
)

// UnvoteInput satisfies the TypedInput interface and represents a unvote transaction.
type UnvoteInput struct {
	UnvoteCommitmentSuffix []byte   // The unconsumed suffix of the output commitment
	Arguments              [][]byte // Witness
	Vote                   []byte   // voter xpub
	SpendCommitment
}

// NewUnvoteInput create a new UnvoteInput struct.
func NewUnvoteInput(arguments [][]byte, sourceID bc.Hash, assetID bc.AssetID, amount, sourcePos uint64, controlProgram []byte, vote []byte) *TxInput {
	sc := SpendCommitment{
		AssetAmount: bc.AssetAmount{
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
		TypedInput: &UnvoteInput{
			SpendCommitment: sc,
			Arguments:       arguments,
			Vote:            vote,
		},
	}
}

// InputType is the interface function for return the input type.
func (ui *UnvoteInput) InputType() uint8 { return SpendInputType }
