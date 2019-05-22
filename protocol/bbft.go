package protocol

import (
	"encoding/hex"
	"time"

	"github.com/vapor/crypto/ed25519"
	edchainkd "github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
	"github.com/vapor/math/checked"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
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
	if err != nil && err != errNotFoundConsensusNode {
		return false, err
	}
	return node != nil, nil
}

func (b *bbft) isIrreversible(block *types.Block) bool {
	signNum, err := b.validateSign(block)
	if err != nil {
		return false
	}

	return signNum > (numOfConsensusNode * 2 / 3)
}

// NextLeaderTime returns the start time of the specified public key as the next leader node
func (b *bbft) NextLeaderTime(pubkey []byte, bestBlockTimestamp, bestBlockHeight uint64) (*time.Time, error) {
	return b.consensusNodeManager.nextLeaderTime(pubkey, bestBlockTimestamp, bestBlockHeight)
}

func (b *bbft) ApplyBlock(voteResultMap map[uint64]*state.VoteResult, block *types.Block) (err error) {
	voteSeq := block.Height / roundVoteBlockNums
	voteResult := voteResultMap[voteSeq]

	if voteResult == nil {
		store := b.consensusNodeManager.store
		voteResult, err = store.GetVoteResult(voteSeq)
		if err != nil && err != ErrNotFoundVoteResult {
			return err
		}
	}

	if voteResult == nil {
		voteResult = &state.VoteResult{
			Seq:             voteSeq,
			NumOfVote:       make(map[string]uint64),
			LastBlockHeight: block.Height,
		}
	}

	voteResultMap[voteSeq] = voteResult

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
	return nil
}

func (b *bbft) DetachBlock(voteResultMap map[uint64]*state.VoteResult, block *types.Block) error {
	voteSeq := block.Height / roundVoteBlockNums
	voteResult := voteResultMap[voteSeq]

	if voteResult == nil {
		store := b.consensusNodeManager.store
		voteResult, err := store.GetVoteResult(voteSeq)
		if err != nil {
			return err
		}
		voteResultMap[voteSeq] = voteResult
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
	return nil
}

// ValidateBlock verify whether the block is valid
func (b *bbft) ValidateBlock(block *types.Block) error {
	signNum, err := b.validateSign(block)
	if err != nil {
		return err
	}

	if signNum == 0 {
		return errors.New("no valid signature")
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
			if b.Hash == block.Hash() {
				continue
			}
			if ok, err := b.BlockWitness.Test(uint32(node.order)); err != nil && ok {
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

// SignBlock signing the block if current node is consensus node
func (b *bbft) SignBlock(block *types.Block) error {
	var xprv edchainkd.XPrv
	switch xpub := xprv.XPub().(type) {
	case edchainkd.XPub:
		node, err := b.consensusNodeManager.getConsensusNode(block.Height, hex.EncodeToString(xpub[:]))
		if err != nil && err != errNotFoundConsensusNode {
			return err
		}

		if node == nil {
			return nil
		}

		block.Witness[node.order] = xprv.Sign(block.Hash().Bytes())
	}
	// // xpub := [64]byte(xprv.XPub())
	// node, err := b.consensusNodeManager.getConsensusNode(block.Height, hex.EncodeToString(xpub[:]))
	// if err != nil && err != errNotFoundConsensusNode {
	// 	return err
	// }

	// if node == nil {
	// 	return nil
	// }

	// block.Witness[node.order] = xprv.Sign(block.Hash().Bytes())
	return nil
}

// UpdateConsensusNodes used to update consensus node after each round of voting
func (b *bbft) UpdateConsensusNodes(blockHeight uint64) error {
	return b.consensusNodeManager.updateConsensusNodes(blockHeight)
}
