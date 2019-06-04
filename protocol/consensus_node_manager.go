package protocol

import (
	"sort"

	"github.com/vapor/config"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
)

var (
	errNotFoundConsensusNode = errors.New("can not found consensus node")
	errNotFoundBlockNode     = errors.New("can not find block node")
)

type consensusNode struct {
	xpub    chainkd.XPub
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

func (c *consensusNodeManager) isBlocker(prevBlockHash *bc.Hash, pubKey string, timeStamp uint64) (bool, error) {
	consensusNodeMap, err := c.getConsensusNodesByVoteResult(prevBlockHash)
	if err != nil {
		return false, err
	}

	consensusNode := consensusNodeMap[pubKey]
	if consensusNode == nil {
		return false, nil
	}

	prevVoteRoundLastBlock, err := c.getPrevRoundVoteLastBlock(prevBlockHash)
	if err != nil {
		return false, err
	}

	startTimestamp := prevVoteRoundLastBlock.Timestamp + consensus.BlockTimeInterval
	begin := getLastBlockTimeInTimeRange(startTimestamp, timeStamp, consensusNode.order, len(consensusNodeMap))
	end := begin + consensus.BlockNumEachNode*consensus.BlockTimeInterval
	return timeStamp >= begin && timeStamp < end, nil
}

func getLastBlockTimeInTimeRange(startTimestamp, endTimestamp, order uint64, numOfConsensusNode int) uint64 {
	// One round of product block time for all consensus nodes
	roundBlockTime := uint64(consensus.BlockNumEachNode * numOfConsensusNode * consensus.BlockTimeInterval)
	// The start time of the last round of product block
	lastRoundStartTime := startTimestamp + (endTimestamp-startTimestamp)/roundBlockTime*roundBlockTime
	// The time of product block of the consensus in last round
	return lastRoundStartTime + order*(consensus.BlockNumEachNode*consensus.BlockTimeInterval)
}

func (c *consensusNodeManager) getPrevRoundVoteLastBlock(prevBlockHash *bc.Hash) (*state.BlockNode, error) {
	prevBlockNode := c.blockIndex.GetNode(prevBlockHash)
	if prevBlockNode == nil {
		return nil, errNotFoundBlockNode
	}

	blockHeight := prevBlockNode.Height + 1

	prevVoteRoundLastBlockHeight := blockHeight/consensus.RoundVoteBlockNums*consensus.RoundVoteBlockNums - 1
	// first round
	if blockHeight/consensus.RoundVoteBlockNums == 0 {
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

	seq := (prevBlockNode.Height + 1) / consensus.RoundVoteBlockNums
	voteResult, err := c.store.GetVoteResult(seq)
	if err != nil {
		// TODO find previous round vote
		voteResult = &state.VoteResult{
			Seq:       seq,
			NumOfVote: make(map[string]uint64),
		}
	}

	lastBlockNode, err := c.getPrevRoundVoteLastBlock(prevBlockHash)
	if err != nil {
		return nil, err
	}

	if err := c.reorganizeVoteResult(voteResult, lastBlockNode); err != nil {
		return nil, err
	}

	if len(voteResult.NumOfVote) == 0 {
		return initConsensusNodes(), nil
	}

	var nodes []*consensusNode
	for pubkey, voteNum := range voteResult.NumOfVote {
		if voteNum >= consensus.MinVoteNum {
			var xpub chainkd.XPub
			if err := xpub.UnmarshalText([]byte(pubkey)); err != nil {
				return nil, err
			}

			nodes = append(nodes, &consensusNode{
				xpub:    xpub,
				voteNum: voteNum,
			})
		}
	}
	// In principle, there is no need to sort all voting nodes.
	// if there is a performance problem, consider the optimization later.
	// TODO not consider the same number of votes
	sort.Sort(consensusNodeSlice(nodes))

	result := make(map[string]*consensusNode)
	for i := 0; i < len(nodes) && i < consensus.NumOfConsensusNode; i++ {
		node := nodes[i]
		node.order = uint64(i)
		result[node.xpub.String()] = node
	}
	return result, nil
}

func (c *consensusNodeManager) reorganizeVoteResult(voteResult *state.VoteResult, forkChainNode *state.BlockNode) error {
	genesisBlockHash := config.GenesisBlock().Hash()
	mainChainNode := c.blockIndex.GetNode(&genesisBlockHash)

	emptyHash := bc.Hash{}
	if voteResult.LastBlockHash != emptyHash {
		mainChainNode = c.blockIndex.GetNode(&voteResult.LastBlockHash)
		if mainChainNode == nil {
			return errNotFoundBlockNode
		}
	}

	var attachNodes []*state.BlockNode
	var detachNodes []*state.BlockNode

	for forkChainNode != nil && mainChainNode != nil && forkChainNode.Hash != mainChainNode.Hash {
		if forkChainNode.Height == mainChainNode.Height {
			detachNodes = append(detachNodes, mainChainNode)
			mainChainNode = mainChainNode.Parent
		}
		attachNodes = append([]*state.BlockNode{forkChainNode}, attachNodes...)
		forkChainNode = forkChainNode.Parent
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
	voteResult, err := c.getVoteResult(voteResultMap, block.Height)
	if err != nil {
		return err
	}

	return voteResult.ApplyBlock(block)
}

func (c *consensusNodeManager) getVoteResult(voteResultMap map[uint64]*state.VoteResult, blockHeight uint64) (*state.VoteResult, error) {
	var err error
	// This round of voting prepares for the next round
	seq := blockHeight/consensus.RoundVoteBlockNums + 1
	voteResult := voteResultMap[seq]
	if blockHeight == 0 {
		voteResult = &state.VoteResult{
			Seq:       seq,
			NumOfVote: make(map[string]uint64),
		}
	}

	if voteResult == nil {
		prevVoteResult := voteResultMap[seq-1]
		if prevVoteResult != nil {
			voteResult = &state.VoteResult{
				Seq:       seq,
				NumOfVote: prevVoteResult.NumOfVote,
			}
		}
	}

	if voteResult == nil {
		voteResult, err = c.store.GetVoteResult(seq)
		if err != nil && err != ErrNotFoundVoteResult {
			return nil, err
		}
	}

	if voteResult == nil {
		voteResult, err = c.store.GetVoteResult(seq - 1)
		if err != nil && err != ErrNotFoundVoteResult {
			return nil, err
		}

		if voteResult != nil {
			voteResult.Seq = seq
			voteResult.LastBlockHash = bc.Hash{}
		}
	}

	if voteResult == nil {
		return nil, errors.New("fail to get vote result")
	}

	voteResultMap[seq] = voteResult
	return voteResult, nil
}

func (c *consensusNodeManager) detachBlock(voteResultMap map[uint64]*state.VoteResult, block *types.Block) error {
	voteSeq := block.Height / consensus.RoundVoteBlockNums
	voteResult := voteResultMap[voteSeq]

	if voteResult == nil {
		voteResult, err := c.store.GetVoteResult(voteSeq)
		if err != nil {
			return err
		}
		voteResultMap[voteSeq] = voteResult
	}

	voteResult.DetachBlock(block)
	return nil
}

func initConsensusNodes() map[string]*consensusNode {
	voteResult := map[string]*consensusNode{}
	for i, xpub := range config.CommonConfig.Federation.Xpubs {
		voteResult[xpub.String()] = &consensusNode{xpub: xpub, voteNum: 0, order: uint64(i)}
	}
	return voteResult
}
