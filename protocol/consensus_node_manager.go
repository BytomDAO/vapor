package protocol

import (
	"encoding/hex"
	"sort"
	"time"

	"github.com/vapor/errors"
	"github.com/vapor/math/checked"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
)

const (
	numOfConsensusNode = 21
	roundVoteBlockNums = 1000

	// BlockTimeInterval indicate product one block per 500 milliseconds
	BlockTimeInterval = 500
	// BlockNumEachNode indicate product three blocks per node in succession
	BlockNumEachNode = 3
)

var (
	errHasNoChanceProductBlock = errors.New("the node has no chance to product a block in this round of voting")
	errNotFoundConsensusNode   = errors.New("can not found consensus node")
	errNotFoundBlockNode       = errors.New("can not find block node")
)

type consensusNode struct {
	pubkey  string
	voteNum uint64
	order   uint64
}

type consensusNodeSlice []*consensusNode

func (c consensusNodeSlice) Len() int           { return len(c) }
func (c consensusNodeSlice) Less(i, j int) bool { return c[i].voteNum > c[j].voteNum }
func (c consensusNodeSlice) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

type consensusNodeManager struct {
	store      Store
	blockIndex *state.BlockIndex
}

func newConsensusNodeManager(store Store, blockIndex *state.BlockIndex) *consensusNodeManager {
	return &consensusNodeManager{
		store:      store,
		blockIndex: blockIndex,
	}
}

func (c *consensusNodeManager) getConsensusNode(blockHash *bc.Hash, pubkey string) (*consensusNode, error) {
	consensusNodeMap, err := c.getConsensusNodesByVoteResult(blockHash)
	if err != nil {
		return nil, err
	}

	node, exist := consensusNodeMap[pubkey]
	if !exist {
		return nil, errNotFoundConsensusNode
	}
	return node, nil
}

func (c *consensusNodeManager) isBlocker(blockHash *bc.Hash, pubkey string) (bool, error) {
	blockNode := c.blockIndex.GetNode(blockHash)
	if blockNode == nil {
		return false, errNotFoundBlockNode
	}

	prevVoteRoundLastBlock, err := c.getPrevRoundVoteLastBlock(blockNode)
	if err != nil {
		return false, err
	}

	startTimestamp := prevVoteRoundLastBlock.Timestamp + BlockTimeInterval

	consensusNode, err := c.getConsensusNode(blockHash, pubkey)
	if err != nil && err != errNotFoundConsensusNode {
		return false, err
	}

	if consensusNode == nil {
		return false, nil
	}

	begin := getLastBlockTimeInTimeRange(startTimestamp, blockNode.Timestamp, consensusNode.order)
	end := begin + BlockNumEachNode*BlockTimeInterval
	return blockNode.Timestamp >= begin && blockNode.Timestamp < end, nil
}

func (c *consensusNodeManager) nextLeaderTimeRange(pubkey []byte, bestBlockHash *bc.Hash) (uint64, uint64, error) {
	bestBlockNode := c.blockIndex.GetNode(bestBlockHash)
	if bestBlockNode == nil {
		return 0, 0, errNotFoundBlockNode
	}

	prevRoundLastBlock, err := c.getPrevRoundVoteLastBlock(bestBlockNode)
	if err != nil {
		return 0, 0, nil
	}

	startTime := prevRoundLastBlock.Timestamp + BlockTimeInterval
	endTime := bestBlockNode.Timestamp + (roundVoteBlockNums-bestBlockNode.Height%roundVoteBlockNums)*BlockTimeInterval

	consensusNode, err := c.getConsensusNode(bestBlockHash, hex.EncodeToString(pubkey))
	if err != nil {
		return 0, 0, err
	}

	nextLeaderTime, err := nextLeaderTimeHelper(startTime, endTime, uint64(time.Now().UnixNano()/1e6), consensusNode.order)
	if err != nil {
		return 0, 0, err
	}

	return nextLeaderTime, nextLeaderTime + BlockNumEachNode*BlockTimeInterval, nil
}

func nextLeaderTimeHelper(startTime, endTime, now, nodeOrder uint64) (uint64, error) {
	nextLeaderTimestamp := getLastBlockTimeInTimeRange(startTime, now, nodeOrder)
	roundBlockTime := uint64(BlockNumEachNode * numOfConsensusNode * BlockTimeInterval)

	if int64(now-nextLeaderTimestamp) >= BlockNumEachNode*BlockTimeInterval {
		nextLeaderTimestamp += roundBlockTime
		if nextLeaderTimestamp >= endTime {
			return 0, errHasNoChanceProductBlock
		}
	}

	return nextLeaderTimestamp, nil
}

func getLastBlockTimeInTimeRange(startTimestamp, endTimestamp, order uint64) uint64 {
	// One round of product block time for all consensus nodes
	roundBlockTime := uint64(BlockNumEachNode * numOfConsensusNode * BlockTimeInterval)
	// The start time of the last round of product block
	lastRoundStartTime := startTimestamp + (endTimestamp-startTimestamp)/roundBlockTime*roundBlockTime
	// The time of product block of the consensus in last round
	return lastRoundStartTime + order*(BlockNumEachNode*BlockTimeInterval)
}

func (c *consensusNodeManager) getPrevRoundVoteLastBlock(blockNode *state.BlockNode) (*state.BlockNode, error) {
	var prevVoteRoundLastBlock *state.BlockNode
	prevVoteRoundLastBlockHeight := blockNode.Height/roundVoteBlockNums*roundVoteBlockNums - 1
	mainChainParent := c.blockIndex.NodeByHeight(blockNode.Height - 1)
	if mainChainParent == nil {
		return nil, errors.New("can not find block of previous height in main chain")
	}

	// block in main chain
	if mainChainParent.Hash == blockNode.Parent.Hash {
		prevVoteRoundLastBlock = c.blockIndex.NodeByHeight(prevVoteRoundLastBlockHeight)
	} else {
		prevBlockNode := blockNode
		for prevBlockNode.Height != prevVoteRoundLastBlockHeight {
			prevBlockNode = c.blockIndex.GetNode(&prevBlockNode.Parent.Hash)
			if prevBlockNode == nil {
				return nil, errNotFoundBlockNode
			}
		}
		prevVoteRoundLastBlock = prevBlockNode
	}
	return prevVoteRoundLastBlock, nil
}

func (c *consensusNodeManager) getConsensusNodesByVoteResult(blockHash *bc.Hash) (map[string]*consensusNode, error) {
	blockNode := c.blockIndex.GetNode(blockHash)
	if blockNode == nil {
		return nil, errNotFoundBlockNode
	}

	seq := blockNode.Height / roundVoteBlockNums
	voteResult, err := c.store.GetVoteResult(seq)
	if err != nil {
		return nil, errors.Wrap(err, "fail to get vote result")
	}

	if !voteResult.Finalized {
		// the vote has not finalized, complement first
		if err := c.complementVoteResult(voteResult); err != nil {
			return nil, err
		}
	}

	mainChainParent := c.blockIndex.NodeByHeight(blockNode.Height - 1)
	if mainChainParent == nil {
		return nil, errors.New("can not find block of previous height in main chain")
	}

	// The block in fork chain, must rollback
	if mainChainParent.Hash != blockNode.Parent.Hash {
		if err := c.rollbackVoteResult(voteResult, blockNode); err != nil {
			return nil, err
		}
	}

	var nodes []*consensusNode
	for pubkey, voteNum := range voteResult.NumOfVote {
		nodes = append(nodes, &consensusNode{
			pubkey:  pubkey,
			voteNum: voteNum,
		})
	}
	// In principle, there is no need to sort all voting nodes.
	// if there is a performance problem, consider the optimization later.
	// TODO not consider the same number of votes
	sort.Sort(consensusNodeSlice(nodes))

	result := make(map[string]*consensusNode)
	for i := 0; i < numOfConsensusNode; i++ {
		node := nodes[i]
		node.order = uint64(i)
		result[node.pubkey] = node
	}
	return result, nil
}

func (c *consensusNodeManager) complementVoteResult(voteResult *state.VoteResult) error {
	lastBlock := c.blockIndex.GetNode(&voteResult.LastBlockHash)
	if lastBlock == nil {
		return errNotFoundBlockNode
	}
	
	for height := lastBlock.Height + 1; height < voteResult.Seq*roundVoteBlockNums; height++ {
		b := c.blockIndex.NodeByHeight(height)
		if b == nil {
			return errNotFoundBlockNode
		}

		block, err := c.store.GetBlock(&b.Hash)
		if err != nil {
			return err
		}
		if err := c.applyBlock(map[uint64]*state.VoteResult{voteResult.Seq: voteResult}, block); err != nil {
			return err
		}
	}
	return nil
}

func (c *consensusNodeManager) rollbackVoteResult(voteResult *state.VoteResult, blockNode *state.BlockNode) error {
	forkChainNode, err := c.getPrevRoundVoteLastBlock(blockNode)
	if err != nil {
		return err
	}

	mainChainNode := c.blockIndex.NodeByHeight(voteResult.Seq*roundVoteBlockNums - 1)
	if mainChainNode == nil {
		return errNotFoundBlockNode
	}

	var attachBlocks []*types.Block
	var detachBlocks []*types.Block
	for forkChainNode.Hash != mainChainNode.Hash {
		attachBlock, err := c.store.GetBlock(&forkChainNode.Hash)
		if err != nil {
			return err
		}

		detachBlock, err := c.store.GetBlock(&mainChainNode.Hash)
		if err != nil {
			return err
		}

		attachBlocks = append([]*types.Block{attachBlock}, attachBlocks...)
		detachBlocks = append(detachBlocks, detachBlock)

		forkChainNode = forkChainNode.Parent
		mainChainNode = mainChainNode.Parent
	}

	for _, block := range detachBlocks {
		if err := c.detachBlock(map[uint64]*state.VoteResult{voteResult.Seq: voteResult}, block); err != nil {
			return err
		}
	}

	for _, block := range attachBlocks {
		if err := c.applyBlock(map[uint64]*state.VoteResult{voteResult.Seq: voteResult}, block); err != nil {
			return err
		}
	}
	return nil
}

func (c *consensusNodeManager) applyBlock(voteResultMap map[uint64]*state.VoteResult, block *types.Block) (err error) {
	voteSeq := block.Height / roundVoteBlockNums
	voteResult := voteResultMap[voteSeq]

	if voteResult == nil {
		voteResult, err = c.store.GetVoteResult(voteSeq)
		if err != nil && err != ErrNotFoundVoteResult {
			return err
		}
	}

	if voteResult == nil {
		voteResult = &state.VoteResult{
			Seq:             voteSeq,
			NumOfVote:       make(map[string]uint64),
			LastBlockHash:   block.Hash(),
		}
	}

	voteResultMap[voteSeq] = voteResult

	if voteResult.LastBlockHash != block.PreviousBlockHash {
		return errors.New("bbft append block error, the block parent hash is not equals last block hash of vote result")
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

	voteResult.Finalized = (block.Height+1)%roundVoteBlockNums == 0
	return nil
}

func (c *consensusNodeManager) detachBlock(voteResultMap map[uint64]*state.VoteResult, block *types.Block) error {
	voteSeq := block.Height / roundVoteBlockNums
	voteResult := voteResultMap[voteSeq]

	if voteResult == nil {
		voteResult, err := c.store.GetVoteResult(voteSeq)
		if err != nil {
			return err
		}
		voteResultMap[voteSeq] = voteResult
	}

	if voteResult.LastBlockHash != block.Hash() {
		return errors.New("bbft detach block error, the block hash is not equals last block hash of vote result")
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

	voteResult.Finalized = false
	return nil
}
