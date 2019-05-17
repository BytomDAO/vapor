package protocol

import (
	"time"
	"encoding/hex"

	"github.com/vapor/crypto/ed25519"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
	"github.com/vapor/protocol/validation"
	"github.com/vapor/crypto/ed25519/chainkd"
)

var (
	errHasNoChanceProductBlock = errors.New("the node has no chance to product a block in this round of voting")
	errBlockSyncNotComplete    = errors.New("current node block synchronization is not complete")
)

type bbft struct {
	consensusNodeManager *consensusNodeManager
}

func newBbft(store Store) *bbft {
	return &bbft{
		consensusNodeManager: newConsensusNodeManager(store),
	}
}

// IsConsensusPubkey determine whether a public key is a consensus node at a specified height
func (b *bbft) IsConsensusPubkey(height uint64, pubkey []byte) (bool, error) {
	node, err := b.consensusNodeManager.getConsensusNode(height, pubkey)
	return node != nil, err
}

// NextLeaderTime returns the start time of the specified public key as the next leader node
func (b *bbft) NextLeaderTime(pubkey []byte, bestBlockHeight, prevRoundLastBlockTimestamp uint64) (*time.Time, error) {
	startTime := prevRoundLastBlockTimestamp*1000 + blockTimeInterval
	consensusNode, err := b.consensusNodeManager.getConsensusNode(bestBlockHeight, pubkey)
	if err != nil {
		return nil, err
	}

	nextLeaderTime, err := nextLeaderTimeHelper(b.consensusNodeManager.effectiveStartHeight, bestBlockHeight, startTime, consensusNode.order)
	if err != nil {
		return nil, err
	}

	if nextLeaderTime.UnixNano() < time.Now().UnixNano() {
		return nil, errBlockSyncNotComplete
	}
	return nextLeaderTime, nil
}

func nextLeaderTimeHelper(startBlockHeight, bestBlockHeight, startTime, nodeOrder uint64) (*time.Time, error) {
	endBlockHeight := startBlockHeight + roundVoteBlockNums
	// exclude genesis block
	if startBlockHeight == 1 {
		endBlockHeight--
	}

	roundBlockNums := uint64(blockNumEachNode * numOfConsensusNode)
	latestRoundBlockHeight := startBlockHeight + (bestBlockHeight-startBlockHeight)/roundBlockNums*roundBlockNums
	nextBlockHeight := latestRoundBlockHeight + blockNumEachNode*nodeOrder

	if int64(bestBlockHeight-nextBlockHeight) >= blockNumEachNode {
		nextBlockHeight += roundBlockNums
		if nextBlockHeight > endBlockHeight {
			return nil, errHasNoChanceProductBlock
		}
	}

	nextLeaderTimestamp := int64(startTime + (nextBlockHeight-startBlockHeight)*blockTimeInterval)
	nextLeaderTime := time.Unix(nextLeaderTimestamp/1000, (nextLeaderTimestamp%1000)*1e6)
	return &nextLeaderTime, nil
}

func (b *bbft) AppendBlock(block *types.Block) error {
	voteSeq := block.Height / roundVoteBlockNums
	store := b.consensusNodeManager.store
	voteResult, err := store.GetVoteResult(voteSeq)
	if err != nil {
		return nil
	}

	if voteResult == nil {
		voteResult = &state.VoteResult{
			Seq: voteSeq,
			NumOfVote: make(map[string]uint64),
			LastBlockHeight: block.Height,
		}
	}

	if voteResult.LastBlockHeight + 1 != block.Height {
		return errors.New("bbft append block error, the block height is not equals last block height plus 1 of vote result")
	}
	
	for _, tx := range block.Transactions {
		for _, output := range tx.Outputs {
			voteOutput, ok := output.TypedOutput.(*types.VoteTxOutput)
			if !ok {
				continue
			}
			pubkey := hex.EncodeToString(voteOutput.Vote)
			voteResult.NumOfVote[pubkey] += voteOutput.Amount
		}
	}

	voteResult.LastBlockHeight++
	voteResult.Finalized = block.Height % roundVoteBlockNums == 0
	return store.SaveVoteResult(voteResult)
}

func (b *bbft) DetachBlock(block *types.Block) error {
	voteSeq := block.Height / roundVoteBlockNums
	store := b.consensusNodeManager.store
	voteResult, err := store.GetVoteResult(voteSeq)
	if err != nil {
		return nil
	}

	if voteResult == nil {
		return nil
	}

	if voteResult.LastBlockHeight != block.Height {
		return errors.New("bbft detach block error, the block height is not equals last block height of vote result")
	}

	for _, tx := range block.Transactions {
		for _, output := range tx.Outputs {
			voteOutput, ok := output.TypedOutput.(*types.VoteTxOutput)
			if !ok {
				continue
			}
			pubkey := hex.EncodeToString(voteOutput.Vote)
			voteResult.NumOfVote[pubkey] -= voteOutput.Amount
		}
	}

	voteResult.LastBlockHeight--
	voteResult.Finalized = false
	return store.SaveVoteResult(voteResult)
}

// ValidateBlock verify whether the block is valid, and return the number of correct signature
func (b *bbft) ValidateBlock(block *types.Block, parent *state.BlockNode) (uint64, error) {
	signNum, err := b.validateSign(block)
	if err != nil {
		return 0, err
	}

	if signNum == 0 {
		return 0, errors.New("invalid block, no valid signature")
	}

	if err := validation.ValidateBlock(types.MapBlock(block), parent); err != nil {
		return 0, err
	}

	if err := b.signBlock(block); err != nil {
		return 0, err
	}
	
	return signNum+1, nil 
}

// validateSign verify the signatures of block, and return the number of correct signature
// if some signature is invalid, they will be reset to nil
func (b *bbft) validateSign(block *types.Block) (uint64, error) {
	var correctSignNum uint64
	blockHeight := block.Height
	consensusNodeMap, err := b.consensusNodeManager.getConsensusNodesByVoteResult(blockHeight / roundVoteBlockNums)
	if err != nil {
		return 0, err
	}

	for pubkey, node := range consensusNodeMap {
		if len(block.Witness) <= int(node.order) {
			continue
		}
		if ed25519.Verify(ed25519.PublicKey(pubkey), block.Hash().Bytes(), block.Witness[node.order]) {
			correctSignNum++
		} else {
			// discard the invalid signature
			block.Witness[node.order] = nil
		}
	}
	return correctSignNum, nil
}

func (b *bbft) signBlock(block *types.Block) error {
	var xprv chainkd.XPrv
	xpub := [64]byte(xprv.XPub())
	node, err := b.consensusNodeManager.getConsensusNode(block.Height, xpub[:])
	if err != nil {
		return err
	}

	if node == nil {
		return nil
	}

	block.Witness[node.order] = xprv.Sign(block.Hash().Bytes())
	return nil
}
