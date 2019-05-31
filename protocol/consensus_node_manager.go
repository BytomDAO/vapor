package protocol

import (
	"encoding/hex"
	"sort"
	"time"

	"github.com/vapor/config"
	"github.com/vapor/errors"
	"github.com/vapor/math/checked"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
)

const (
	NumOfConsensusNode = 21
	roundVoteBlockNums = 1000

	// BlockTimeInterval indicate product one block per 500 milliseconds
	BlockTimeInterval = 500
	// BlockNumEachNode indicate product three blocks per node in succession
	BlockNumEachNode = 3
)

var (
	errNotFoundConsensusNode = errors.New("can not found consensus node")
	errNotFoundBlockNode     = errors.New("can not find block node")
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

func (c *consensusNodeManager) getConsensusNode(prevBlockHash *bc.Hash, pubkey string) (*consensusNode, error) {
	consensusNodeMap, err := c.getConsensusNodesByVoteResult(prevBlockHash)
	if err != nil {
		return nil, err
	}

	node, exist := consensusNodeMap[pubkey]
	if !exist {
		return nil, errNotFoundConsensusNode
	}
	return node, nil
}

func (c *consensusNodeManager) isBlocker(block *types.Block, pubKey string) (bool, error) {
	consensusNode, err := c.getConsensusNode(&block.PreviousBlockHash, pubKey)
	if err != nil && err != errNotFoundConsensusNode {
		return false, err
	}

	if consensusNode == nil {
		return false, nil
	}

	prevVoteRoundLastBlock, err := c.getPrevRoundVoteLastBlock(&block.PreviousBlockHash)
	if err != nil {
		return false, err
	}

	startTimestamp := prevVoteRoundLastBlock.Timestamp + BlockTimeInterval

	begin := getLastBlockTimeInTimeRange(startTimestamp, block.Timestamp, consensusNode.order)
	end := begin + BlockNumEachNode*BlockTimeInterval
	return block.Timestamp >= begin && block.Timestamp < end, nil
}

func (c *consensusNodeManager) nextLeaderTimeRange(pubkey []byte, bestBlockHash *bc.Hash) (uint64, uint64, error) {
	bestBlockNode := c.blockIndex.GetNode(bestBlockHash)
	if bestBlockNode == nil {
		return 0, 0, errNotFoundBlockNode
	}

	consensusNode, err := c.getConsensusNode(&bestBlockNode.Parent.Hash, hex.EncodeToString(pubkey))
	if err != nil {
		return 0, 0, err
	}

	prevRoundLastBlock, err := c.getPrevRoundVoteLastBlock(&bestBlockNode.Parent.Hash)
	if err != nil {
		return 0, 0, err
	}

	startTime := prevRoundLastBlock.Timestamp + BlockTimeInterval

	nextLeaderTime, err := nextLeaderTimeHelper(startTime, uint64(time.Now().UnixNano()/1e6), consensusNode.order)
	if err != nil {
		return 0, 0, err
	}

	return nextLeaderTime, nextLeaderTime + BlockNumEachNode*BlockTimeInterval, nil
}

func nextLeaderTimeHelper(startTime, now, nodeOrder uint64) (uint64, error) {
	nextLeaderTimestamp := getLastBlockTimeInTimeRange(startTime, now, nodeOrder)
	roundBlockTime := uint64(BlockNumEachNode * NumOfConsensusNode * BlockTimeInterval)

	if now > nextLeaderTimestamp {
		nextLeaderTimestamp += roundBlockTime
	}

	return nextLeaderTimestamp, nil
}

func getLastBlockTimeInTimeRange(startTimestamp, endTimestamp, order uint64) uint64 {
	// One round of product block time for all consensus nodes
	roundBlockTime := uint64(BlockNumEachNode * NumOfConsensusNode * BlockTimeInterval)
	// The start time of the last round of product block
	lastRoundStartTime := startTimestamp + (endTimestamp-startTimestamp)/roundBlockTime*roundBlockTime
	// The time of product block of the consensus in last round
	return lastRoundStartTime + order*(BlockNumEachNode*BlockTimeInterval)
}

func (c *consensusNodeManager) getPrevRoundVoteLastBlock(prevBlockHash *bc.Hash) (*state.BlockNode, error) {
	prevBlockNode := c.blockIndex.GetNode(prevBlockHash)
	if prevBlockNode == nil {
		return nil, errNotFoundBlockNode
	}

	blockHeight := prevBlockNode.Height + 1

	prevVoteRoundLastBlockHeight := blockHeight/roundVoteBlockNums*roundVoteBlockNums - 1
	// first round
	if blockHeight/roundVoteBlockNums == 0 {
		prevVoteRoundLastBlockHeight = 0
	}

	lastBlockNode := prevBlockNode.GetParent(prevVoteRoundLastBlockHeight)
	if lastBlockNode == nil {
		return nil, errNotFoundBlockNode
	}
	return lastBlockNode, nil
}

func (c *consensusNodeManager) getConsensusNodesByVoteResult(prevBlockHash *bc.Hash) (map[string]*consensusNode, error) {
	prevBlockNode := c.blockIndex.GetNode(prevBlockHash)
	if prevBlockNode == nil {
		return nil, errNotFoundBlockNode
	}

	seq := (prevBlockNode.Height + 1) / roundVoteBlockNums
	if seq == 0 {
		return initVoteResult(), nil
	}

	voteResult, err := c.store.GetVoteResult(seq)
	if err != nil {
		// fail to find vote result, try to construct
		voteResult = &state.VoteResult{
			Seq:       seq,
			NumOfVote: make(map[string]uint64),
			Finalized: false,
		}
	}

	lastBlockNode, err := c.getPrevRoundVoteLastBlock(prevBlockHash)
	if err != nil {
		return nil, err
	}

	if err := c.reorganizeVoteResult(voteResult, lastBlockNode); err != nil {
		return nil, err
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
	for i := 0; i < len(nodes) && i < NumOfConsensusNode; i++ {
		node := nodes[i]
		node.order = uint64(i)
		result[node.pubkey] = node
	}
	return result, nil
}

func (c *consensusNodeManager) reorganizeVoteResult(voteResult *state.VoteResult, forkChainNode *state.BlockNode) error {
	var mainChainNode *state.BlockNode
	emptyHash := bc.Hash{}
	if voteResult.LastBlockHash != emptyHash {
		mainChainNode = c.blockIndex.GetNode(&voteResult.LastBlockHash)
		if mainChainNode == nil {
			return errNotFoundBlockNode
		}
	}

	var attachNodes []*state.BlockNode
	var detachNodes []*state.BlockNode

	for forkChainNode.Hash != mainChainNode.Hash && forkChainNode.Height >= (voteResult.Seq-1)*roundVoteBlockNums {
		attachNodes = append([]*state.BlockNode{forkChainNode}, attachNodes...)
		forkChainNode = forkChainNode.Parent

		if mainChainNode != nil && forkChainNode.Height == mainChainNode.Height {
			detachNodes = append(detachNodes, mainChainNode)
			mainChainNode = mainChainNode.Parent
		}
	}

	for _, node := range detachNodes {
		block, err := c.store.GetBlock(&node.Hash)
		if err != nil {
			return err
		}

		if err := c.detachBlock(map[uint64]*state.VoteResult{voteResult.Seq: voteResult}, block); err != nil {
			return err
		}
	}

	for _, node := range attachNodes {
		block, err := c.store.GetBlock(&node.Hash)
		if err != nil {
			return err
		}

		if err := c.applyBlock(map[uint64]*state.VoteResult{voteResult.Seq: voteResult}, block); err != nil {
			return err
		}
	}
	return nil
}

func (c *consensusNodeManager) applyBlock(voteResultMap map[uint64]*state.VoteResult, block *types.Block) (err error) {
	voteSeq := block.Height / roundVoteBlockNums
	voteResult, err := c.getVoteResult(voteResultMap, voteSeq)
	if err != nil {
		return err
	}

	emptyHash := bc.Hash{}
	if voteResult.LastBlockHash != emptyHash && voteResult.LastBlockHash != block.PreviousBlockHash {
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

func (c *consensusNodeManager) getVoteResult(voteResultMap map[uint64]*state.VoteResult, seq uint64) (*state.VoteResult, error) {
	var err error
	voteResult := voteResultMap[seq]
	if voteResult == nil {
		prevVoteResult := voteResultMap[seq - 1]
		voteResult = &state.VoteResult {
			Seq: seq,
			NumOfVote: prevVoteResult.NumOfVote,
			Finalized: false,
		}
	}

	if voteResult == nil {
		voteResult, err = c.store.GetVoteResult(seq)
		if err != nil && err != ErrNotFoundVoteResult {
			return nil, err
		}
	}

	if voteResult == nil {
		voteResult, err := c.store.GetVoteResult(seq - 1)
		if err != nil && err != ErrNotFoundVoteResult {
			return nil, err
		}
		// previous round voting must have finalized
		if !voteResult.Finalized {
			return nil, errors.New("previous round voting has not finalized")
		}

		voteResult.Finalized = false
		voteResult.LastBlockHash = bc.Hash{}
	}
	voteResultMap[seq] = voteResult
	return voteResult, nil
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

func initVoteResult() map[string]*consensusNode {
	voteResult := map[string]*consensusNode{}
	for i, pubkey := range config.CommonConfig.Federation.Xpubs {
		pubkeyStr := pubkey.String()
		voteResult[pubkeyStr] = &consensusNode{pubkey: pubkeyStr, voteNum: 0, order: uint64(i)}
	}
	return voteResult
}
