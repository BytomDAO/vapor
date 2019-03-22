package types

import (
	"github.com/vapor/protocol/bc"
)

type DposTx struct {
	SpendCommitmentSuffix []byte
	Type                  TxType
	From                  string
	To                    string
	Amount                uint64
	Stake                 uint64
	Arguments             [][]byte
	Info                  string
	SpendCommitment
}

func NewDpos(arguments [][]byte, from, to string, sourceID bc.Hash, assetID bc.AssetID, stake, amount, sourcePos uint64, controlProgram []byte, t TxType, height uint64) *TxInput {
	var vote string

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
		TypedInput: &DposTx{
			SpendCommitment: sc,
			Type:            t,
			Amount:          amount,
			Arguments:       arguments,
			Info:            vote,
			Stake:           stake,
			From:            from,
			To:              to,
		},
	}
}

// InputType is the interface function for return the input type.
func (si *DposTx) InputType() uint8 { return DposInputType }
