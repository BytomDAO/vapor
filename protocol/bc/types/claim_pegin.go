package types

import (
	"github.com/bytom/protocol/bc"
)

type ClaimCommitment struct {
	bc.AssetAmount
	VMVersion      uint64
	ControlProgram []byte
}

// ClaimInput satisfies the TypedInput interface and represents a spend transaction.
type ClaimInput struct {
	SpendCommitmentSuffix []byte   // The unconsumed suffix of the output commitment
	Arguments             [][]byte // Witness
	SpendCommitment
}

// NewClaimInputInput create a new SpendInput struct.
func NewClaimInputInput(arguments [][]byte, sourceID bc.Hash, assetID bc.AssetID, amount, sourcePos uint64, controlProgram []byte) *TxInput {

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
		},
	}
}

// InputType is the interface function for return the input type.
func (si *ClaimInput) InputType() uint8 { return ClainPeginInputType }
