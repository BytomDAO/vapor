package common

import (
	"sort"

	"github.com/bytom/vapor/api"
	"github.com/bytom/vapor/consensus"
)

const NumOfBPNode = 42

func CalcStandByNodes(voteResult []*api.VoteInfo) []*api.VoteInfo {
	sort.Slice(voteResult, func(i, j int) bool {
		return voteResult[i].VoteNum > voteResult[j].VoteNum
	})

	result := []*api.VoteInfo{}
	for i := int(consensus.ActiveNetParams.NumOfConsensusNode); i < NumOfBPNode && i < len(voteResult); i++ {
		if voteResult[i].VoteNum < consensus.ActiveNetParams.MinConsensusNodeVoteNum {
			break
		}
		result = append(result, voteResult[i])
	}
	return result
}
