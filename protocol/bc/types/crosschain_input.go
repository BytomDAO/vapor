package types

import (
	"github.com/vapor/protocol/bc"
)

// CrossChainInput satisfies the TypedInput interface and represents a cross-chain transaction.
type CrossChainInput struct {
	MainchainSourceID       bc.Hash
	MainchainSourcePosition uint64
	AssetAmount             bc.AssetAmount
	ControlProgram          []byte
	SpendCommitmentSuffix   []byte   // compatible with main-chain's types.SpendInput
	Arguments               [][]byte // Witness
}

// NewCrossChainInput create a new CrossChainInput struct.
// The source is created/issued by trusted federation and hence there is no need
// to refer to it.
func NewCrossChainInput(sourceID bc.Hash, sourcePos uint64, assetID bc.AssetID, amount uint64, controlProgram []byte, arguments [][]byte) *TxInput {
	return &TxInput{
		AssetVersion: 1,
		TypedInput: &CrossChainInput{
			MainchainSourceID:       sourceID,
			MainchainSourcePosition: sourcePos,
			AssetAmount: bc.AssetAmount{
				AssetId: &assetID,
				Amount:  amount,
			},
			ControlProgram: controlProgram,
			Arguments:      arguments,
		},
	}
}

// InputType is the interface function for return the input type.
func (si *CrossChainInput) InputType() uint8 { return CrossChainInputType }
