package consensusreward

import (
	"fmt"

	"github.com/vapor/api"
	"github.com/vapor/consensus"
)

func PickStandbyBPVoteResult(voteResult []api.VoteInfo) ([]api.VoteInfo, error) {
	low := int(consensus.ActiveNetParams.NumOfConsensusNode)
	high := len(voteResult) - 1
	position := 0
	minConsensusNodeVoteNum := consensus.ActiveNetParams.MinConsensusNodeVoteNum
	if high < 0 {
		return nil, fmt.Errorf("Vote result is empty")
	}
	if high < int(consensus.ActiveNetParams.NumOfConsensusNode) || voteResult[low].VoteNum < minConsensusNodeVoteNum {
		return nil, fmt.Errorf("No Standby BP Node")
	}
	for low <= high {
		mid := low + (high-low)/2
		if voteResult[mid].VoteNum >= minConsensusNodeVoteNum {
			low = mid + 1
			if voteResult[low].VoteNum < minConsensusNodeVoteNum {
				position = low
				break
			}
		} else {
			high = mid - 1
			if voteResult[high].VoteNum >= minConsensusNodeVoteNum {
				position = mid
				break
			}
		}
	}
	return voteResult[int(consensus.ActiveNetParams.NumOfConsensusNode):position], nil
}
