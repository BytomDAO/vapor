package protocol

import (
	"encoding/hex"
	"sort"
	"sync"

	"github.com/vapor/errors"
)

const (
	numOfConsensusNode = 21
	roundVoteBlockNums = 1000

	// product one block per 500 milliseconds
	blockTimeInterval = 500
	blockNumEachNode  = 3
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
	sync.RWMutex
}

func newConsensusNodeManager(store Store) *consensusNodeManager {
	return &consensusNodeManager{
		consensusNodeMap:     make(map[string]*consensusNode),
		effectiveStartHeight: 1,
		store:                store,
	}
}

func (c *consensusNodeManager) getConsensusNode(height uint64, pubkey []byte) (*consensusNode, error) {
	defer c.RUnlock()
	c.RLock()
	if height >= c.effectiveStartHeight + roundVoteBlockNums {
		return nil, errors.New("the vote has not been completed for the specified block height ")
	}

	var err error
	consensusNodeMap := c.consensusNodeMap
	// query history vote result
	if height < c.effectiveStartHeight {
		consensusNodeMap, err = c.getConsensusNodesByVoteResult(height / roundVoteBlockNums)
		if err != nil {
			return nil, err
		}
	}

	encodePubkey := hex.EncodeToString(pubkey)
	return consensusNodeMap[encodePubkey], nil
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

	if voteResult == nil {
		return nil, errors.New("can not find vote result by given vote sequence")
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
