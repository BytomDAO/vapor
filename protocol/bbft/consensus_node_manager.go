package bbft

import (
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/vapor/database"
	"github.com/vapor/errors"
	"github.com/vapor/protocol"
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
	store                *database.Store
	chain                *protocol.Chain
	sync.RWMutex
}

func newConsensusNodeManager(store *database.Store, chain *protocol.Chain) *consensusNodeManager {
	return &consensusNodeManager{
		consensusNodeMap:     make(map[string]*consensusNode),
		effectiveStartHeight: 1,
		chain:                chain,
		store:                store,
	}
}

func (c *consensusNodeManager) isConsensusPubkey(height uint64, pubkey []byte) (bool, error) {
	defer c.RUnlock()
	c.RLock()
	if height >= c.effectiveStartHeight + roundVoteBlockNums {
		return false, errors.New("the vote has not been completed for the specified block height ")
	}

	var err error
	consensusNodeMap := c.consensusNodeMap
	// query history vote result
	if height < c.effectiveStartHeight {
		consensusNodeMap, err = c.getConsensusNodesByVoteResult(height / roundVoteBlockNums)
		if err != nil {
			return false, err
		}
	}

	encodePubkey := hex.EncodeToString(pubkey)
	_, exist := consensusNodeMap[encodePubkey]
	return exist, nil
}

func (c *consensusNodeManager) nextLeaderTime(pubkey []byte) (*time.Time, error) {
	defer c.RLock()
	c.RLock()

	encodePubkey := hex.EncodeToString(pubkey)
	consensusNode, ok := c.consensusNodeMap[encodePubkey]
	if !ok {
		return nil, fmt.Errorf("pubkey:%s is not consensus node", encodePubkey)
	}

	startBlockHeight := c.effectiveStartHeight
	bestBlockHeight := c.chain.BestBlockHeight()
	
	prevRoundLastBlock, err := c.chain.GetHeaderByHeight(startBlockHeight - 1)
	if err != nil {
		return nil, err
	}

	startTime := prevRoundLastBlock.Timestamp * 1000 + blockTimeInterval
	return nextLeaderTimeHelper(startBlockHeight, bestBlockHeight, startTime, consensusNode.order)
}

func nextLeaderTimeHelper(startBlockHeight, bestBlockHeight, startTime, nodeOrder uint64) (*time.Time, error) {
	endBlockHeight := startBlockHeight + roundVoteBlockNums
	// exclude genesis block
	if startBlockHeight == 1 {
		endBlockHeight--
	}

	roundBlockNums := uint64(blockNumEachNode * numOfConsensusNode)
	latestRoundBlockHeight := startBlockHeight + (bestBlockHeight - startBlockHeight) / roundBlockNums * roundBlockNums
	nextBlockHeight := latestRoundBlockHeight + blockNumEachNode * nodeOrder

	if int64(bestBlockHeight - nextBlockHeight) >= blockNumEachNode {
		nextBlockHeight += roundBlockNums
		if nextBlockHeight > endBlockHeight {
			return nil, errHasNoChanceProductBlock
		}
	}

	nextLeaderTimestamp := int64(startTime + (nextBlockHeight - startBlockHeight) * blockTimeInterval)
	nextLeaderTime := time.Unix(nextLeaderTimestamp / 1000, (nextLeaderTimestamp % 1000) * 1e6)
	return &nextLeaderTime, nil
}

// UpdateConsensusNodes used to update consensus node after each round of voting
func (c *consensusNodeManager) UpdateConsensusNodes(voteSeq uint64) error {
	defer c.Unlock()
	c.Lock()
	if voteSeq <= c.effectiveStartHeight / roundVoteBlockNums {
		return nil
	}

	consensusNodeMap, err := c.getConsensusNodesByVoteResult(voteSeq)
	if err != nil {
		return err
	}

	c.consensusNodeMap = consensusNodeMap
	c.effectiveStartHeight = voteSeq * roundVoteBlockNums
	return nil
}

func (c *consensusNodeManager) getConsensusNodesByVoteResult(voteSeq uint64) (map[string]*consensusNode, error) {
	voteResult, err := c.store.GetVoteResult(voteSeq)
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
	for i, node := range nodes {
		node.order = uint64(i)
		result[node.pubkey] = node
	}
	return result, nil
}
