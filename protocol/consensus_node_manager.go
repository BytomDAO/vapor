package protocol

import (
	"github.com/vapor/config"
	"github.com/vapor/consensus"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/state"
)

var (
	errNotFoundConsensusNode = errors.New("can not found consensus node")
	errNotFoundBlockNode     = errors.New("can not find block node")
)

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

func (c *consensusNodeManager) getConsensusNode(prevBlockHash *bc.Hash, pubkey string) (*state.ConsensusNode, error) {
	consensusNodeMap, err := c.getConsensusNodes(prevBlockHash)
	if err != nil {
		return nil, err
	}

	node, exist := consensusNodeMap[pubkey]
	if !exist {
		return nil, errNotFoundConsensusNode
	}
	return node, nil
}

func (c *consensusNodeManager) isBlocker(prevBlockHash *bc.Hash, pubKey string, timeStamp uint64) (bool, error) {
	consensusNodeMap, err := c.getConsensusNodes(prevBlockHash)
	if err != nil {
		return false, err
	}

	consensusNode := consensusNodeMap[pubKey]
	if consensusNode == nil {
		return false, nil
	}

	prevVoteRoundLastBlock, err := c.getPrevRoundLastBlock(prevBlockHash)
	if err != nil {
		return false, err
	}

	startTimestamp := prevVoteRoundLastBlock.Timestamp + consensus.BlockTimeInterval
	begin := getLastBlockTimeInTimeRange(startTimestamp, timeStamp, consensusNode.Order, uint64(len(consensusNodeMap)))
	end := begin + consensus.BlockNumEachNode*consensus.BlockTimeInterval
	return timeStamp >= begin && timeStamp < end, nil
}

func getLastBlockTimeInTimeRange(startTimestamp, endTimestamp, order, numOfConsensusNode uint64) uint64 {
	// One round of product block time for all consensus nodes
	roundBlockTime := consensus.BlockNumEachNode * numOfConsensusNode * consensus.BlockTimeInterval
	// The start time of the last round of product block
	lastRoundStartTime := startTimestamp + (endTimestamp-startTimestamp)/roundBlockTime*roundBlockTime
	// The time of product block of the consensus in last round
	return lastRoundStartTime + order*(consensus.BlockNumEachNode*consensus.BlockTimeInterval)
}

func (c *consensusNodeManager) getPrevRoundLastBlock(prevBlockHash *bc.Hash) (*state.BlockNode, error) {
	node := c.blockIndex.GetNode(prevBlockHash)
	if node == nil {
		return nil, errNotFoundBlockNode
	}

	for node.Height%consensus.RoundVoteBlockNums != 0 {
		node = node.Parent
	}
	return node, nil
}

func (c *consensusNodeManager) getConsensusNodes(prevBlockHash *bc.Hash) (map[string]*state.ConsensusNode, error) {
	prevBlockNode := c.blockIndex.GetNode(prevBlockHash)
	if prevBlockNode == nil {
		return nil, errNotFoundBlockNode
	}

	preSeq := state.CalcVoteSeq(prevBlockNode.Height + 1) - 1
	if bestSeq := state.CalcVoteSeq(c.blockIndex.BestNode().Height); preSeq > bestSeq {
		preSeq = bestSeq
	}

	voteResult, err := c.store.GetVoteResult(preSeq)
	if err != nil {
		return nil, err
	}

	lastBlockNode, err := c.getPrevRoundLastBlock(prevBlockHash)
	if err != nil {
		return nil, err
	}

	if err := c.reorganizeVoteResult(voteResult, lastBlockNode); err != nil {
		return nil, err
	}

	if len(voteResult.NumOfVote) == 0 {
		return federationNodes(), nil
	}
	return voteResult.ConsensusNodes()
}

func (c *consensusNodeManager) getBestVoteResult() (*state.VoteResult, error) {
	blockNode := c.blockIndex.BestNode()
	seq := state.CalcVoteSeq(blockNode.Height)
	voteResult, err := c.store.GetVoteResult(seq)
	if err != nil {
		return nil, err
	}

	if err := c.reorganizeVoteResult(voteResult, blockNode); err != nil {
		return nil, err
	}

	return voteResult, nil
}

func (c *consensusNodeManager) reorganizeVoteResult(voteResult *state.VoteResult, node *state.BlockNode) error {
	mainChainNode := c.blockIndex.GetNode(&voteResult.BlockHash)
	var attachNodes []*state.BlockNode
	var detachNodes []*state.BlockNode
	for forkChainNode := node; mainChainNode != forkChainNode; node = node.Parent {
		if forkChainNode.Height == mainChainNode.Height {
			detachNodes = append(detachNodes, mainChainNode)
			mainChainNode = mainChainNode.Parent
		}
		attachNodes = append([]*state.BlockNode{forkChainNode}, attachNodes...)
	}

	for _, node := range detachNodes {
		block, err := c.store.GetBlock(&node.Hash)
		if err != nil {
			return err
		}

		if err := voteResult.DetachBlock(block); err != nil {
			return err
		}
	}

	for _, node := range attachNodes {
		block, err := c.store.GetBlock(&node.Hash)
		if err != nil {
			return err
		}

		if err := voteResult.ApplyBlock(block); err != nil {
			return err
		}
	}
	return nil
}

func federationNodes() map[string]*state.ConsensusNode {
	voteResult := map[string]*state.ConsensusNode{}
	for i, xpub := range config.CommonConfig.Federation.Xpubs {
		voteResult[xpub.String()] = &state.ConsensusNode{XPub: xpub, VoteNum: 0, Order: uint64(i)}
	}
	return voteResult
}
