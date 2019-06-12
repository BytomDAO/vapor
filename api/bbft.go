package api

import (
	"sort"

	chainjson "github.com/vapor/encoding/json"
)

type voteInfo struct {
	PubKey      string `json:"pub_key"`
	VoteNum     uint64 `json:"vote_num"`
}

type voteInfoSlice []*voteInfo
func (v voteInfoSlice) Len() int { return len(v) }
func (v voteInfoSlice) Less(i, j int) bool { return v[i].VoteNum > v[j].VoteNum }
func (v voteInfoSlice) Swap(i, j int) { v[i], v[j] = v[j], v[i] }

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

	voteResult, err := a.chain.GetVoteResultByHash(&blockHash)
	if err != nil {
		return NewErrorResponse(err)
	}

	voteInfos := []*voteInfo{}
	for pubKey, voteNum := range voteResult.NumOfVote {
		voteInfos = append(voteInfos, &voteInfo{
			PubKey:      pubKey,
			VoteNum:     voteNum,
		})
	}
	sort.Sort(voteInfoSlice(voteInfos))
	return NewSuccessResponse(voteInfos)
}
