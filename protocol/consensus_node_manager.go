package protocol

import (
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
	store    Store
	bestNode *state.BlockNode
}

func newConsensusNodeManager(store Store, bestNode *state.BlockNode) *consensusNodeManager {
	return &consensusNodeManager{
		store:    store,
		bestNode: bestNode,
	}
}

func (c *consensusNodeManager) getBestNode() *state.BlockNode {
	return c.bestNode
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

func (c *consensusNodeManager) getBlocker(prevBlockHash *bc.Hash, timeStamp uint64) (string, error) {
	consensusNodeMap, err := c.getConsensusNodes(prevBlockHash)
	if err != nil {
		return "", err
	}

	prevVoteRoundLastBlock, err := c.getPrevRoundLastBlock(prevBlockHash)
	if err != nil {
		return "", err
	}

	startTimestamp := prevVoteRoundLastBlock.Timestamp + consensus.BlockTimeInterval
	order := getBlockerOrder(startTimestamp, timeStamp, uint64(len(consensusNodeMap)))
	for xPub, consensusNode := range consensusNodeMap {
		if consensusNode.Order == order {
			return xPub, nil
		}
	}

	// impossible occur
	return "", errors.New("can not find blocker by given timestamp")
}

func getBlockerOrder(startTimestamp, blockTimestamp, numOfConsensusNode uint64) uint64 {
	// One round of product block time for all consensus nodes
	roundBlockTime := consensus.BlockNumEachNode * numOfConsensusNode * consensus.BlockTimeInterval
	// The start time of the last round of product block
	lastRoundStartTime := startTimestamp + (blockTimestamp-startTimestamp)/roundBlockTime*roundBlockTime
	// Order of blocker
	return (blockTimestamp - lastRoundStartTime) / (consensus.BlockNumEachNode * consensus.BlockTimeInterval)
}

func (c *consensusNodeManager) getPrevRoundLastBlock(prevBlockHash *bc.Hash) (*state.BlockNode, error) {
	node, err := c.store.GetBlockNode(prevBlockHash)
	if err != nil {
		return nil, errNotFoundBlockNode
	}

	for node.Height%consensus.RoundVoteBlockNums != 0 {
		node, err = c.store.GetBlockNode(node.Parent)
		if err != nil {
			return nil, err
		}
	}
	return node, nil
}

func (c *consensusNodeManager) getConsensusNodes(prevBlockHash *bc.Hash) (map[string]*state.ConsensusNode, error) {
	prevBlockNode, err := c.store.GetBlockNode(prevBlockHash)
	if err != nil {
		return nil, errNotFoundBlockNode
	}

	bestNode := c.getBestNode()
	preSeq := state.CalcVoteSeq(prevBlockNode.Height+1) - 1
	if bestSeq := state.CalcVoteSeq(bestNode.Height); preSeq > bestSeq {
		preSeq = bestSeq
	}

	lastBlockNode, err := c.getPrevRoundLastBlock(prevBlockHash)
	if err != nil {
		return nil, err
	}

	voteResult, err := c.getVoteResult(preSeq, lastBlockNode)
	if err != nil {
		return nil, err
	}

	return voteResult.ConsensusNodes()
}

func (c *consensusNodeManager) getBestVoteResult() (*state.VoteResult, error) {
	bestNode := c.getBestNode()
	seq := state.CalcVoteSeq(bestNode.Height)
	return c.getVoteResult(seq, bestNode)
}

// getVoteResult return the vote result
// seq represent the sequence of vote
// blockNode represent the chain in which the result of the vote is located
// Voting results need to be adjusted according to the chain
func (c *consensusNodeManager) getVoteResult(seq uint64, blockNode *state.BlockNode) (*state.VoteResult, error) {
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
	mainChainNode, err := c.store.GetBlockNode(&voteResult.BlockHash)
	if err != nil {
		return errNotFoundBlockNode
	}

	var attachNodes []*state.BlockNode
	var detachNodes []*state.BlockNode
	for forkChainNode := node; mainChainNode != forkChainNode; {
		var forChainRollback, mainChainRollBack bool
		if forChainRollback = forkChainNode.Height >= mainChainNode.Height; forChainRollback {
			attachNodes = append([]*state.BlockNode{forkChainNode}, attachNodes...)
		}
		if mainChainRollBack = forkChainNode.Height <= mainChainNode.Height; mainChainRollBack {
			detachNodes = append(detachNodes, mainChainNode)
		}
		if forChainRollback {
			forkChainNode, err = c.store.GetBlockNode(forkChainNode.Parent)
			if err != nil {
				return errNotFoundBlockNode
			}
		}
		if mainChainRollBack {
			mainChainNode, err = c.store.GetBlockNode(mainChainNode.Parent)
			if err != nil {
				return errNotFoundBlockNode
			}
		}
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
