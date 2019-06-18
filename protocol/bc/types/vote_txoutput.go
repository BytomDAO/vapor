package types

import (
	"github.com/vapor/protocol/bc"
)

// VoteTxOutput satisfies the TypedOutput interface and represents a vote transaction.
type VoteTxOutput struct {
	OutputCommitment
	// Unconsumed suffixes of the commitment and witness extensible strings.
	CommitmentSuffix []byte
	Vote             []byte
}

// NewVoteOutput create a new output struct
func NewVoteOutput(assetID bc.AssetID, amount uint64, controlProgram []byte, vote []byte) *TxOutput {
	return &TxOutput{
		AssetVersion: 1,
		TypedOutput: &VoteTxOutput{
			OutputCommitment: OutputCommitment{
				AssetAmount: bc.AssetAmount{
					AssetId: &assetID,
					Amount:  amount,
				},
				VMVersion:      1,
				ControlProgram: controlProgram,
			},
			Vote: vote,
		},
	}
}

func (it *VoteTxOutput) OutputType() uint8 { return VoteOutputType }
