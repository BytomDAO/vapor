package api

import (
	chainjson "github.com/vapor/encoding/json"
)

type voteInfo struct {
	PubKey      string `json:"pub_key"`
	VoteNum     uint64 `json:"vote_num"`
	IsConsensus bool   `json:"is_consensus"`
}

func (a *API) getVoteResult(req struct {
	BlockHash   chainjson.HexBytes `json:"block_hash"`
	BlockHeight uint64             `json:"block_height"`
}) Response {
	blockHash := hexBytesToHash(req.BlockHash)
	if len(req.BlockHash) != 32 {
		blockNode, err := a.chain.GetHeaderByHeight(req.BlockHeight)
		if err != nil {
			return NewErrorResponse(err)
		}

		blockHash = blockNode.Hash()
	}

	voteResult, err := a.chain.GetVoteResultByHash(&blockHash)
	if err != nil {
		return NewErrorResponse(err)
	}

	consensusNodes, err := voteResult.ConsensusNodes()
	if err != nil {
		return NewErrorResponse(err)
	}

	voteInfos := []*voteInfo{}
	for pubKey, voteNum := range voteResult.NumOfVote {
		_, isConsensus := consensusNodes[pubKey]
		voteInfos = append(voteInfos, &voteInfo{
			PubKey:      pubKey,
			VoteNum:     voteNum,
			IsConsensus: isConsensus,
		})
	}
	return NewSuccessResponse(voteInfos)
}
