package state

import (
	"encoding/hex"

	"github.com/vapor/errors"
	"github.com/vapor/math/checked"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

var errVotingOperationOverFlow = errors.New("voting operation result overflow")

// VoteResult represents a snapshot of each round of DPOS voting
// Seq indicates the sequence of current votes, which start from zero
// NumOfVote indicates the number of votes each consensus node receives, the key of map represent public key
// Finalized indicates whether this vote is finalized
type VoteResult struct {
	Seq           uint64
	NumOfVote     map[string]uint64
	LastBlockHash bc.Hash
}

func (v *VoteResult) ApplyBlock(block *types.Block) error {
	if v.LastBlockHash != block.PreviousBlockHash {
		return errors.New("block parent hash is not equals last block hash of vote result")
	}

	for _, tx := range block.Transactions {
		for _, input := range tx.Inputs {
			unVoteInput, ok := input.TypedInput.(*types.UnvoteInput)
			if !ok {
				continue
			}

			pubkey := hex.EncodeToString(unVoteInput.Vote)
			v.NumOfVote[pubkey], ok = checked.SubUint64(v.NumOfVote[pubkey], unVoteInput.Amount)
			if !ok {
				return errVotingOperationOverFlow
			}

			if v.NumOfVote[pubkey] == 0 {
				delete(v.NumOfVote, pubkey)
			}
		}

		for _, output := range tx.Outputs {
			voteOutput, ok := output.TypedOutput.(*types.VoteTxOutput)
			if !ok {
				continue
			}

			pubkey := hex.EncodeToString(voteOutput.Vote)
			if v.NumOfVote[pubkey], ok = checked.AddUint64(v.NumOfVote[pubkey], voteOutput.Amount); !ok {
				return errVotingOperationOverFlow
			}
		}
	}

	return nil
}

func (v *VoteResult) DetachBlock(block *types.Block) error {
	if v.LastBlockHash != block.Hash() {
		return errors.New("block hash is not equals last block hash of vote result")
	}

	for _, tx := range block.Transactions {
		for _, input := range tx.Inputs {
			unVoteInput, ok := input.TypedInput.(*types.UnvoteInput)
			if !ok {
				continue
			}

			pubkey := hex.EncodeToString(unVoteInput.Vote)
			if v.NumOfVote[pubkey], ok = checked.AddUint64(v.NumOfVote[pubkey], unVoteInput.Amount); !ok {
				return errVotingOperationOverFlow
			}
		}

		for _, output := range tx.Outputs {
			voteOutput, ok := output.TypedOutput.(*types.VoteTxOutput)
			if !ok {
				continue
			}

			pubkey := hex.EncodeToString(voteOutput.Vote)
			v.NumOfVote[pubkey], ok = checked.SubUint64(v.NumOfVote[pubkey], voteOutput.Amount)
			if !ok {
				return errVotingOperationOverFlow
			}

			if v.NumOfVote[pubkey] == 0 {
				delete(v.NumOfVote, pubkey)
			}
		}
	}

	return nil
}
