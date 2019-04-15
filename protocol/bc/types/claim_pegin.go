package types

import (
	"github.com/vapor/protocol/bc"
)

// ClaimInput satisfies the TypedInput interface and represents a spend transaction.
type ClaimInput struct {
	SpendCommitmentSuffix []byte   // The unconsumed suffix of the output commitment
	Arguments             [][]byte // Witness
	AssetDefinition       []byte
	SpendCommitment
}

// NewClaimInput create a new SpendInput struct.
func NewClaimInput(arguments [][]byte, sourceID bc.Hash, assetID bc.AssetID, amount, sourcePos uint64, controlProgram []byte, assetDefinition []byte) *TxInput {

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
		TypedInput: &ClaimInput{
			SpendCommitment: sc,
			Arguments:       arguments,
			AssetDefinition: assetDefinition,
		},
	}
}

// InputType is the interface function for return the input type.
func (si *ClaimInput) InputType() uint8 { return ClainPeginInputType }
