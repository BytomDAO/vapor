package state

import (
	"encoding/hex"
	"sort"

	"github.com/vapor/config"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
	"github.com/vapor/math/checked"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

var errVotingOperationOverFlow = errors.New("voting operation result overflow")

type ConsensusNode struct {
	XPub    chainkd.XPub
	VoteNum uint64
	Order   uint64
}

type byVote []*ConsensusNode

func (c byVote) Len() int { return len(c) }
func (c byVote) Less(i, j int) bool {
	return c[i].VoteNum > c[j].VoteNum || (c[i].VoteNum == c[j].VoteNum && c[i].XPub.String() > c[j].XPub.String())
}
func (c byVote) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

// seq 0 is the genesis block
// seq 1 is the the block height 1, to block height RoundVoteBlockNums
// seq 2 is the block height RoundVoteBlockNums + 1 to block height 2 * RoundVoteBlockNums
// consensus node of the current round is the final result of previous round
func CalcVoteSeq(blockHeight uint64) uint64 {
	if blockHeight == 0 {
		return 0
	}
	return (blockHeight-1)/consensus.RoundVoteBlockNums + 1
}

// VoteResult represents a snapshot of each round of DPOS voting
// Seq indicates the sequence of current votes, which start from zero
// NumOfVote indicates the number of votes each consensus node receives, the key of map represent public key
// Finalized indicates whether this vote is finalized
type VoteResult struct {
	Seq         uint64
	NumOfVote   map[string]uint64
	BlockHash   bc.Hash
	BlockHeight uint64
}

func (v *VoteResult) ApplyBlock(block *types.Block) error {
	if v.BlockHash != block.PreviousBlockHash {
		return errors.New("block parent hash is not equals last block hash of vote result")
	}

	for _, tx := range block.Transactions {
		for _, input := range tx.Inputs {
			vetoInput, ok := input.TypedInput.(*types.VetoInput)
			if !ok {
				continue
			}

			pubkey := hex.EncodeToString(vetoInput.Vote)
			v.NumOfVote[pubkey], ok = checked.SubUint64(v.NumOfVote[pubkey], vetoInput.Amount)
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

	v.BlockHash = block.Hash()
	v.BlockHeight = block.Height
	v.Seq = CalcVoteSeq(block.Height)
	return nil
}

func (v *VoteResult) ConsensusNodes() (map[string]*ConsensusNode, error) {
	var nodes []*ConsensusNode
	for pubkey, voteNum := range v.NumOfVote {
		if voteNum >= consensus.MinConsensusNodeVoteNum {
			var xpub chainkd.XPub
			if err := xpub.UnmarshalText([]byte(pubkey)); err != nil {
				return nil, err
			}

			nodes = append(nodes, &ConsensusNode{XPub: xpub, VoteNum: voteNum})
		}
	}
	// In principle, there is no need to sort all voting nodes.
	// if there is a performance problem, consider the optimization later.
	sort.Sort(byVote(nodes))
	result := make(map[string]*ConsensusNode)
	for i := 0; i < len(nodes) && i < consensus.NumOfConsensusNode; i++ {
		nodes[i].Order = uint64(i)
		result[nodes[i].XPub.String()] = nodes[i]
	}

	if len(result) != 0 {
		return result, nil
	}
	return federationNodes(), nil
}

func federationNodes() map[string]*ConsensusNode {
	voteResult := map[string]*ConsensusNode{}
	for i, xpub := range config.CommonConfig.Federation.Xpubs {
		voteResult[xpub.String()] = &ConsensusNode{XPub: xpub, VoteNum: 0, Order: uint64(i)}
	}
	return voteResult
}

func (v *VoteResult) DetachBlock(block *types.Block) error {
	if v.BlockHash != block.Hash() {
		return errors.New("block hash is not equals last block hash of vote result")
	}

	for i := len(block.Transactions) - 1; i >= 0; i-- {
		tx := block.Transactions[i]
		for _, input := range tx.Inputs {
			vetoInput, ok := input.TypedInput.(*types.VetoInput)
			if !ok {
				continue
			}

			pubkey := hex.EncodeToString(vetoInput.Vote)
			if v.NumOfVote[pubkey], ok = checked.AddUint64(v.NumOfVote[pubkey], vetoInput.Amount); !ok {
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

	v.BlockHash = block.PreviousBlockHash
	v.BlockHeight = block.Height - 1
	v.Seq = CalcVoteSeq(block.Height - 1)
	return nil
}

func (v *VoteResult) Fork() *VoteResult {
	f := &VoteResult{
		Seq:         v.Seq,
		NumOfVote:   map[string]uint64{},
		BlockHash:   v.BlockHash,
		BlockHeight: v.BlockHeight,
	}

	for key, value := range v.NumOfVote {
		f.NumOfVote[key] = value
	}
	return f
}

func (v *VoteResult) IsFinalize() bool {
	return v.BlockHeight%consensus.RoundVoteBlockNums == 0
}
