package protocol

import (
	"encoding/hex"
	"time"

	"github.com/vapor/crypto/ed25519"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
	"github.com/vapor/math/checked"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
	"github.com/vapor/protocol/validation"
)

var (
	errVotingOperationOverFlow = errors.New("voting operation result overflow")
)

type bbft struct {
	consensusNodeManager *consensusNodeManager
}

func newBbft(store Store, blockIndex *state.BlockIndex) *bbft {
	return &bbft{
		consensusNodeManager: newConsensusNodeManager(store, blockIndex),
	}
}

// IsConsensusPubkey determine whether a public key is a consensus node at a specified height
func (b *bbft) IsConsensusPubkey(height uint64, pubkey []byte) (bool, error) {
	node, err := b.consensusNodeManager.getConsensusNode(height, hex.EncodeToString(pubkey))
	return node != nil, err
}

func (b *bbft) isIrreversible(block *types.Block) bool {
	signNum, err := b.validateSign(block)
	if err != nil {
		return false
	}

	return signNum > (numOfConsensusNode / 3 * 2)
}

// NextLeaderTime returns the start time of the specified public key as the next leader node
func (b *bbft) NextLeaderTime(pubkey []byte) (*time.Time, error) {
	return b.consensusNodeManager.nextLeaderTime(pubkey)
}

func (b *bbft) AppendBlock(block *types.Block) error {
	voteSeq := block.Height / roundVoteBlockNums
	store := b.consensusNodeManager.store
	voteResult, err := store.GetVoteResult(voteSeq)
	if err != nil && err != ErrNotFoundVoteResult {
		return nil
	}

	if voteResult == nil {
		voteResult = &state.VoteResult{
			Seq:             voteSeq,
			NumOfVote:       make(map[string]uint64),
			LastBlockHeight: block.Height,
		}
	}

	if voteResult.LastBlockHeight+1 != block.Height {
		return errors.New("bbft append block error, the block height is not equals last block height plus 1 of vote result")
	}

	for _, tx := range block.Transactions {
		for _, input := range tx.Inputs {
			unVoteInput, ok := input.TypedInput.(*types.UnvoteInput)
			if !ok {
				continue
			}
			
			pubkey := hex.EncodeToString(unVoteInput.Vote)
			voteResult.NumOfVote[pubkey], ok = checked.SubUint64(voteResult.NumOfVote[pubkey], unVoteInput.Amount)
			if !ok {
				return errVotingOperationOverFlow
			}
		}
		for _, output := range tx.Outputs {
			voteOutput, ok := output.TypedOutput.(*types.VoteTxOutput)
			if !ok {
				continue
			}

			pubkey := hex.EncodeToString(voteOutput.Vote)
			voteResult.NumOfVote[pubkey], ok = checked.AddUint64(voteResult.NumOfVote[pubkey], voteOutput.Amount)
			if !ok {
				return errVotingOperationOverFlow
			}
		}
	}

	voteResult.LastBlockHeight++
	voteResult.Finalized = (block.Height+1)%roundVoteBlockNums == 0
	return store.SaveVoteResult(voteResult)
}

func (b *bbft) DetachBlock(block *types.Block) error {
	voteSeq := block.Height / roundVoteBlockNums
	store := b.consensusNodeManager.store
	voteResult, err := store.GetVoteResult(voteSeq)
	if err != nil {
		return nil
	}

	if voteResult.LastBlockHeight != block.Height {
		return errors.New("bbft detach block error, the block height is not equals last block height of vote result")
	}

	for _, tx := range block.Transactions {
		for _, input := range tx.Inputs {
			unVoteInput, ok := input.TypedInput.(*types.UnvoteInput)
			if !ok {
				continue
			}
			
			pubkey := hex.EncodeToString(unVoteInput.Vote)
			voteResult.NumOfVote[pubkey], ok = checked.AddUint64(voteResult.NumOfVote[pubkey], unVoteInput.Amount)
			if !ok {
				return errVotingOperationOverFlow
			}
		}
		for _, output := range tx.Outputs {
			voteOutput, ok := output.TypedOutput.(*types.VoteTxOutput)
			if !ok {
				continue
			}
			
			pubkey := hex.EncodeToString(voteOutput.Vote)
			voteResult.NumOfVote[pubkey], ok = checked.SubUint64(voteResult.NumOfVote[pubkey], voteOutput.Amount)
			if !ok {
				return errVotingOperationOverFlow
			}
		}
	}

	voteResult.LastBlockHeight--
	voteResult.Finalized = false
	return store.SaveVoteResult(voteResult)
}

// ValidateBlock verify whether the block is valid
func (b *bbft) ValidateBlock(block *types.Block, parent *state.BlockNode) error {
	signNum, err := b.validateSign(block)
	if err != nil {
		return err
	}

	if signNum == 0 {
		return errors.New("no valid signature")
	}

	if err := validation.ValidateBlock(types.MapBlock(block), parent); err != nil {
		return err
	}

	if err := b.signBlock(block); err != nil {
		return err
	}

	return nil
}

// validateSign verify the signatures of block, and return the number of correct signature
// if some signature is invalid, they will be reset to nil
// if the block has not the signature of blocker, it will return error
func (b *bbft) validateSign(block *types.Block) (uint64, error) {
	var correctSignNum uint64
	consensusNodeMap, err := b.consensusNodeManager.getConsensusNodesByVoteResult(block.Height)
	if err != nil {
		return 0, err
	}

	hasBlockerSign := false
	for pubkey, node := range consensusNodeMap {
		if len(block.Witness) <= int(node.order) {
			continue
		}

		blocks := b.consensusNodeManager.blockIndex.NodesByHeight(block.Height)
		for _, b := range blocks {
			if b.Hash != block.Hash() && b.BlockWitness.Test(uint(node.order)) {
				// Consensus node is signed twice with the same block height, discard the signature
				block.Witness[node.order] = nil
				break
			}
		}

		if ed25519.Verify(ed25519.PublicKey(pubkey), block.Hash().Bytes(), block.Witness[node.order]) {
			correctSignNum++
			isBlocker, err := b.consensusNodeManager.isBlocker(block.Height, block.Timestamp, pubkey)
			if err != nil {
				return 0, err
			}
			if isBlocker {
				hasBlockerSign = true
			}
		} else {
			// discard the invalid signature
			block.Witness[node.order] = nil
		}
	}
	if !hasBlockerSign {
		return 0, errors.New("the block has no signature of the blocker")
	}
	return correctSignNum, nil
}

func (b *bbft) signBlock(block *types.Block) error {
	var xprv chainkd.XPrv
	xpub := [64]byte(xprv.XPub())
	node, err := b.consensusNodeManager.getConsensusNode(block.Height, hex.EncodeToString(xpub[:]))
	if err != nil {
		return err
	}

	if node == nil {
		return nil
	}

	block.Witness[node.order] = xprv.Sign(block.Hash().Bytes())
	return nil
}
