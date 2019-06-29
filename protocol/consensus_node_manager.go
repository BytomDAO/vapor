package protocol

import (
	"github.com/vapor/consensus"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
)

var (
	errNotFoundConsensusNode = errors.New("can not found consensus node")
	errNotFoundBlockNode     = errors.New("can not find block node")
)

func (c *Chain) getConsensusNode(prevBlockHash *bc.Hash, pubkey string) (*state.ConsensusNode, error) {
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

// GetBlocker return blocker by specified timestamp
func (c *Chain) GetBlocker(prevBlockHash *bc.Hash, timeStamp uint64) (string, error) {
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

func (c *Chain) getPrevRoundLastBlock(prevBlockHash *bc.Hash) (*types.BlockHeader, error) {
	blockHeader, err := c.store.GetBlockHeader(prevBlockHash)
	if err != nil {
		return nil, errNotFoundBlockNode
	}

	for blockHeader.Height%consensus.RoundVoteBlockNums != 0 {
		blockHeader, err = c.store.GetBlockHeader(&blockHeader.PreviousBlockHash)
		if err != nil {
			return nil, err
		}
	}
	return blockHeader, nil
}

func (c *Chain) getConsensusNodes(prevBlockHash *bc.Hash) (map[string]*state.ConsensusNode, error) {
	prevBlockHeader, err := c.store.GetBlockHeader(prevBlockHash)
	if err != nil {
		return nil, errNotFoundBlockNode
	}

	bestBlockHeader := c.bestBlockHeader
	preSeq := state.CalcVoteSeq(prevBlockHeader.Height+1) - 1
	if bestSeq := state.CalcVoteSeq(bestBlockHeader.Height); preSeq > bestSeq {
		preSeq = bestSeq
	}

	lastBlockHeader, err := c.getPrevRoundLastBlock(prevBlockHash)
	if err != nil {
		return nil, err
	}

	consensusResult, err := c.getConsensusResult(preSeq, lastBlockHeader)
	if err != nil {
		return nil, err
	}

	return consensusResult.ConsensusNodes()
}

func (c *Chain) getBestConsensusResult() (*state.ConsensusResult, error) {
	bestBlockHeader := c.bestBlockHeader
	seq := state.CalcVoteSeq(bestBlockHeader.Height)
	return c.getConsensusResult(seq, bestBlockHeader)
}

// getConsensusResult return the vote result
// seq represent the sequence of vote
// blockHeader represent the chain in which the result of the vote is located
// Voting results need to be adjusted according to the chain
func (c *Chain) getConsensusResult(seq uint64, blockHeader *types.BlockHeader) (*state.ConsensusResult, error) {
	consensusResult, err := c.store.GetConsensusResult(seq)
	if err != nil {
		return nil, err
	}

	if err := c.reorganizeConsensusResult(consensusResult, blockHeader); err != nil {
		return nil, err
	}

	return consensusResult, nil
}

func (c *Chain) reorganizeConsensusResult(consensusResult *state.ConsensusResult, blockHeader *types.BlockHeader) error {
	mainChainBlockHeader, err := c.store.GetBlockHeader(&consensusResult.BlockHash)
	if err != nil {
		return err
	}

	attachBlockHeaders, detachBlockHeaders, err := c.calcReorganizeChain(blockHeader, mainChainBlockHeader)
	if err != nil {
		return err
	}

	for _, bh := range detachBlockHeaders {
		blockHash := bh.Hash()
		block, err := c.store.GetBlock(&blockHash)
		if err != nil {
			return err
		}

		if err := consensusResult.DetachBlock(block); err != nil {
			return err
		}
	}

	for _, bh := range attachBlockHeaders {
		blockHash := bh.Hash()
		block, err := c.store.GetBlock(&blockHash)
		if err != nil {
			return err
		}

		if err := consensusResult.ApplyBlock(block); err != nil {
			return err
		}
	}
	return nil
}
