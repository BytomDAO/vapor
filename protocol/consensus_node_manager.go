package protocol

import (
	"fmt"
	"time"
	"sort"
	"sync"
	"encoding/hex"

	"github.com/vapor/errors"
	"github.com/vapor/protocol/state"
)

const (
	numOfConsensusNode = 21
	roundVoteBlockNums = 1000

	// product one block per 500 milliseconds
	blockTimeInterval = 500
	blockNumEachNode  = 3
)

var (
	errHasNoChanceProductBlock = errors.New("the node has no chance to product a block in this round of voting")
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
	consensusNodeMap     map[string]*consensusNode
	effectiveStartHeight uint64
	store                Store
	blockIndex           *state.BlockIndex
	sync.RWMutex
}

func newConsensusNodeManager(store Store, blockIndex *state.BlockIndex) *consensusNodeManager {
	return &consensusNodeManager{
		consensusNodeMap:     make(map[string]*consensusNode),
		effectiveStartHeight: 1,
		store:                store,
		blockIndex:           blockIndex,
	}
}

func (c *consensusNodeManager) getConsensusNode(height uint64, pubkey string) (*consensusNode, error) {
	defer c.RUnlock()
	c.RLock()
	if height >= c.effectiveStartHeight + roundVoteBlockNums {
		return nil, errors.New("the vote has not been completed for the specified block height ")
	}

	var err error
	consensusNodeMap := c.consensusNodeMap
	// query history vote result
	if height < c.effectiveStartHeight {
		consensusNodeMap, err = c.getConsensusNodesByVoteResult(height)
		if err != nil {
			return nil, err
		}
	}

	return consensusNodeMap[pubkey], nil
}

func (c *consensusNodeManager) isBlocker(height uint64, blockTimestamp uint64, pubkey string) (bool, error) {
	prevVoteRoundLastBlock := c.blockIndex.NodeByHeight(height - 1)
	startTimestamp := prevVoteRoundLastBlock.Timestamp + blockTimeInterval
	
	consensusNodeMap, err := c.getConsensusNodesByVoteResult(height)
	if err != nil {
		return false, err
	}

	blockerNode, exist := consensusNodeMap[pubkey]
	if !exist {
		return false, nil
	}

	lastRoundStartTime := (blockTimestamp - startTimestamp) / numOfConsensusNode * numOfConsensusNode
	begin := lastRoundStartTime + blockerNode.order * (blockNumEachNode * blockTimeInterval)
	end := begin + blockNumEachNode * blockTimeInterval
	return blockTimestamp >= begin && blockTimestamp < end, nil
}

func (c *consensusNodeManager) nextLeaderTime(pubkey []byte, bestBlockTimestamp, bestBlockHeight uint64) (*time.Time, error) {
	defer c.RUnlock()
	c.RLock()

	startHeight := c.effectiveStartHeight
	prevRoundLastBlock := c.blockIndex.NodeByHeight(startHeight - 1)
	startTime := prevRoundLastBlock.Timestamp + blockTimeInterval
	endTime := bestBlockTimestamp + (roundVoteBlockNums - bestBlockHeight % roundVoteBlockNums) * blockTimeInterval
	
	consensusNode, exist := c.consensusNodeMap[hex.EncodeToString(pubkey)]
	if !exist {
		return nil, fmt.Errorf("pubkey:%s is not consensus node", hex.EncodeToString(pubkey))
	}

	nextLeaderTime, err := nextLeaderTimeHelper(startTime, endTime, uint64(time.Now().UnixNano() / 1e6), consensusNode.order)
	if err != nil {
		return nil, err
	}

	return nextLeaderTime, nil
}

func nextLeaderTimeHelper(startTime, endTime, now, nodeOrder uint64) (*time.Time, error) {
	roundBlockTime := uint64(blockNumEachNode * numOfConsensusNode * blockTimeInterval)
	latestRoundBeginTime := startTime + ((now - startTime) / roundBlockTime) * roundBlockTime
	nextLeaderTimestamp := latestRoundBeginTime + (blockNumEachNode * blockTimeInterval) * nodeOrder

	if int64(now - nextLeaderTimestamp) >= blockNumEachNode * blockTimeInterval {
		nextLeaderTimestamp += roundBlockTime
		if nextLeaderTimestamp >= endTime {
			return nil, errHasNoChanceProductBlock
		}
	}

	nextLeaderTime := time.Unix(int64(nextLeaderTimestamp) / 1000, (int64(nextLeaderTimestamp) % 1000) * 1e6)
	return &nextLeaderTime, nil
}

// UpdateConsensusNodes used to update consensus node after each round of voting
func (c *consensusNodeManager) UpdateConsensusNodes(blockHeight uint64) error {
	defer c.Unlock()
	c.Lock()
	if blockHeight <= c.effectiveStartHeight {
		return nil
	}

	consensusNodeMap, err := c.getConsensusNodesByVoteResult(blockHeight)
	if err != nil {
		return err
	}

	c.consensusNodeMap = consensusNodeMap
	c.effectiveStartHeight = blockHeight / roundVoteBlockNums * roundVoteBlockNums
	return nil
}

func (c *consensusNodeManager) getConsensusNodesByVoteResult(blockHeight uint64) (map[string]*consensusNode, error) {
	defer c.RUnlock()
	c.RLock()
	if blockHeight >= c.effectiveStartHeight + roundVoteBlockNums {
		return nil, errors.New("the given block height is greater than current vote start height")
	}

	if blockHeight >= c.effectiveStartHeight {
		return c.consensusNodeMap, nil
	}

	voteResult, err := c.store.GetVoteResult(blockHeight / roundVoteBlockNums)
	if err != nil {
		return nil, err
	}

	if !voteResult.Finalized {
		return nil, errors.New("vote result is not finalized")
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
