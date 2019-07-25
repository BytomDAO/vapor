package consensusreward

import (
	"sort"

	"github.com/vapor/api"
	"github.com/vapor/consensus"
)

func PickStandbyBPVoteResult(voteResult []api.VoteInfo) []api.VoteInfo {
	numOfConsensusNode := int(consensus.ActiveNetParams.NumOfConsensusNode)
	minConsensusNodeVoteNum := consensus.ActiveNetParams.MinConsensusNodeVoteNum

	sort.Slice(voteResult, func(i, j int) bool {
		return voteResult[i].VoteNum > voteResult[j].VoteNum
	})
	if len(voteResult) <= int(numOfConsensusNode) || voteResult[numOfConsensusNode].VoteNum < minConsensusNodeVoteNum {
		return nil
	}

	position := sort.Search(len(voteResult[numOfConsensusNode:]), func(i int) bool {
		return voteResult[i].VoteNum < minConsensusNodeVoteNum
	})
	return voteResult[int(consensus.ActiveNetParams.NumOfConsensusNode):position]
}
