package api

import (
	"sort"

	chainjson "github.com/bytom/vapor/encoding/json"
)

type VoteInfo struct {
	Vote    string `json:"vote"`
	VoteNum uint64 `json:"vote_number"`
}

type voteInfoSlice []*VoteInfo

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

	voteInfos := []*VoteInfo{}
	for pubKey, voteNum := range consensusResult.NumOfVote {
		voteInfos = append(voteInfos, &VoteInfo{
			Vote:    pubKey,
			VoteNum: voteNum,
		})
	}
	sort.Sort(voteInfoSlice(voteInfos))
	return NewSuccessResponse(voteInfos)
}
