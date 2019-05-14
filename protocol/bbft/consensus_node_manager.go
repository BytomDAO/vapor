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

type consensusNodeManager struct {
	consensusNodes       []*consensusNode
	effectiveStartHeight uint64
	store                *database.Store
	chain                *protocol.Chain
	sync.RWMutex
}

func newConsensusNodeManager(store *database.Store) *consensusNodeManager {
	return &consensusNodeManager{
		consensusNodes:       []*consensusNode{},
		effectiveStartHeight: 1,
		store:                store,
	}
}

type consensusNode struct {
	pubkey  string
	voteNum uint64
}

type consensusNodeSlice []*consensusNode

func (c consensusNodeSlice) Len() int           { return len(c) }
func (c consensusNodeSlice) Less(i, j int) bool { return c[i].voteNum > c[j].voteNum }
func (c consensusNodeSlice) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

func (c *consensusNodeManager) isConsensusPubkey(height uint64, pubkey []byte) (bool, error) {
	defer c.RUnlock()
	c.RLock()
	if height >= c.effectiveStartHeight + roundVoteBlockNums {
		return false, errors.New("the vote has not been completed for the specified block height ")
	}

	var err error
	var consensusNodes []*consensusNode
	if height >= c.effectiveStartHeight {
		consensusNodes = c.consensusNodes
	// query history vote result
	} else if height < c.effectiveStartHeight {
		consensusNodes, err = c.getConsensusNodesByVoteResult(height / roundVoteBlockNums)
		if err != nil {
			return false, err
		}
	}

	encodePubkey := hex.EncodeToString(pubkey)
	for _, node := range consensusNodes {
		if node.pubkey == encodePubkey {
			return true, nil
		}
	}
	return false, nil
}

func (c *consensusNodeManager) nextLeaderTime(pubkey []byte) (*time.Time, error) {
	defer c.RLock()
	c.RLock()

	prevRoundLastBlockHeight := c.effectiveStartHeight - 1
	prevRoundLastBlock, err := c.chain.GetBlockByHeight(prevRoundLastBlockHeight)
	if err != nil {
		return nil, err
	}

	// The timestamp of the block can only be accurate to the second, so take the ceil of timestamp
	begin := (int64(prevRoundLastBlock.Timestamp) + 1) * 1000
	now := time.Now().UnixNano() / 1e6
	roundVoteTime := int64(roundVoteBlockNums * blockTimeInterval)

	if now - begin >= roundVoteTime {
		return nil, fmt.Errorf("the node has not completed block synchronization")
	}

	roundBlockTime := int64(blockNumEachNode * numOfConsensusNode * blockTimeInterval)
	latestRoundBeginTime := begin + ((now - begin) / roundBlockTime) * roundBlockTime

	encodePubkey := hex.EncodeToString(pubkey)
	var nodeSeq int64 = -1
	for i, node := range c.consensusNodes {
		if node.pubkey == encodePubkey {
			nodeSeq = int64(i)
		}
	}
	if nodeSeq == -1 {
		return nil, fmt.Errorf("pubkey:%s is not consensus node", encodePubkey)
	}

	nextLeaderTimestamp := latestRoundBeginTime + (blockNumEachNode * blockTimeInterval) * nodeSeq
	if now - nextLeaderTimestamp >= 0 {
		nextLeaderTimestamp += roundBlockTime
		if nextLeaderTimestamp - begin > roundVoteTime {
			return nil, fmt.Errorf("pubkey:%s has no chance to product a block in this round of voting", pubkey)
		}
	}
	
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

	consensusNodes, err := c.getConsensusNodesByVoteResult(voteSeq)
	if err != nil {
		return err
	}

	c.consensusNodes = consensusNodes
	c.effectiveStartHeight = voteSeq * roundVoteBlockNums
	return nil
}

func (c *consensusNodeManager) getConsensusNodesByVoteResult(voteSeq uint64) ([]*consensusNode, error) {
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

	return nodes[0:numOfConsensusNode], nil
}
