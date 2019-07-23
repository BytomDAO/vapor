package apinode

import (
	"encoding/json"

	"github.com/vapor/api"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc/types"
)

func (n *Node) GetBlockByHash(hash string) (*types.Block, error) {
	return n.getRawBlock(&getRawBlockReq{BlockHash: hash})
}

func (n *Node) GetBlockByHeight(height uint64) (*types.Block, error) {
	return n.getRawBlock(&getRawBlockReq{BlockHeight: height})
}

type getRawBlockReq struct {
	BlockHeight uint64 `json:"block_height"`
	BlockHash   string `json:"block_hash"`
}

func (n *Node) getRawBlock(req *getRawBlockReq) (*types.Block, error) {
	url := "/get-raw-block"
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	resp := &api.GetRawBlockResp{}
	return resp.RawBlock, n.request(url, payload, resp)
}

func (n *Node) GetVoteByHash(hash string) ([]voteInfo, error) {
	return n.getVoteResult(&getVoteResultReq{BlockHash: hash})
}

func (n *Node) GetVoteByHeight(height uint64) ([]voteInfo, error) {
	return n.getVoteResult(&getVoteResultReq{BlockHeight: height})
}

type getVoteResultReq struct {
	BlockHeight uint64 `json:"block_height"`
	BlockHash   string `json:"block_hash"`
}

type voteInfo struct {
	Vote    string `json:"vote"`
	VoteNum uint64 `json:"vote_number"`
}

func (n *Node) getVoteResult(req *getVoteResultReq) ([]voteInfo, error) {
	url := "/get-vote-result"
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}
	resp := &[]voteInfo{}
	return *resp, n.request(url, payload, resp)
}
