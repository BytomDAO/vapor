package api

import (
	"sort"

	chainjson "github.com/vapor/encoding/json"
)

type voteInfo struct {
	Vote    string `json:"vote"`
	VoteNum uint64 `json:"vote_number"`
}

type voteInfoSlice []*voteInfo

func (v voteInfoSlice) Len() int           { return len(v) }
func (v voteInfoSlice) Less(i, j int) bool { return v[i].VoteNum > v[j].VoteNum }
func (v voteInfoSlice) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }

func (a *API) getVoteResult(req struct {
	BlockHash   chainjson.HexBytes `json:"block_hash"`
	BlockHeight uint64             `json:"block_height"`
}) Response {
	blockHash := hexBytesToHash(req.BlockHash)
	if len(req.BlockHash) != 32 {
		blockHeader, err := a.chain.GetHeaderByHeight(req.BlockHeight)
		if err != nil {
			return NewErrorResponse(err)
		}

		blockHash = blockHeader.Hash()
	}

	consensusResult, err := a.chain.GetConsensusResultByHash(&blockHash)
	if err != nil {
		return NewErrorResponse(err)
	}

	voteInfos := []*voteInfo{}
	for pubKey, voteNum := range consensusResult.NumOfVote {
		voteInfos = append(voteInfos, &voteInfo{
			Vote:    pubKey,
			VoteNum: voteNum,
		})
	}
	sort.Sort(voteInfoSlice(voteInfos))
	return NewSuccessResponse(voteInfos)
}
