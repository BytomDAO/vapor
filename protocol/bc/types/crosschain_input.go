package types

import (
	"github.com/bytom/vapor/protocol/bc"
)

// CrossChainInput satisfies the TypedInput interface and represents a cross-chain transaction.
type CrossChainInput struct {
	SpendCommitmentSuffix []byte   // The unconsumed suffix of the spend commitment
	Arguments             [][]byte // Witness
	SpendCommitment

	AssetDefinition   []byte
	IssuanceVMVersion uint64
	IssuanceProgram   []byte
}

// NewCrossChainInput create a new CrossChainInput struct.
// The source is created/issued by trusted federation and hence there is no need
// to refer to it.
func NewCrossChainInput(arguments [][]byte, sourceID bc.Hash, assetID bc.AssetID, amount, sourcePos, IssuanceVMVersion uint64, assetDefinition, issuanceProgram []byte) *TxInput {
	sc := SpendCommitment{
		AssetAmount: bc.AssetAmount{
			AssetId: &assetID,
			Amount:  amount,
		},
		SourceID:       sourceID,
		SourcePosition: sourcePos,
		VMVersion:      1,
	}
	return &TxInput{
		AssetVersion: 1,
		TypedInput: &CrossChainInput{
			SpendCommitment:   sc,
			Arguments:         arguments,
			AssetDefinition:   assetDefinition,
			IssuanceVMVersion: IssuanceVMVersion,
			IssuanceProgram:   issuanceProgram,
		},
	}
}

// InputType is the interface function for return the input type.
func (si *CrossChainInput) InputType() uint8 { return CrossChainInputType }
