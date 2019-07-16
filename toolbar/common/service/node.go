package service

import (
	"encoding/json"

	"github.com/vapor/errors"
	"github.com/vapor/toolbar/federation/util"
	"github.com/vapor/protocol/bc"
)

// Node can invoke the api which provide by the full node server
type Node struct {
	hostPort string
}

// Node create a api client with target server
func NewNode(hostPort string) *Node {
	return &Node{hostPort: hostPort}
}

func (n *Node) GetBlockByHash(hash string) (string, *bc.TransactionStatus, error) {
	return n.getRawBlock(&getRawBlockReq{BlockHash: hash})
}

func (n *Node) GetBlockByHeight(height uint64) (string, *bc.TransactionStatus, error) {
	return n.getRawBlock(&getRawBlockReq{BlockHeight: height})
}

type getBlockCountResp struct {
	BlockCount uint64 `json:"block_count"`
}

func (n *Node) GetBlockCount() (uint64, error) {
	url := "/get-block-count"
	res := &getBlockCountResp{}
	return res.BlockCount, n.request(url, nil, res)
}

type getRawBlockReq struct {
	BlockHeight uint64 `json:"block_height"`
	BlockHash   string `json:"block_hash"`
}

type getRawBlockResp struct {
	RawBlock string `json:"raw_block"`
	// TransactionStatus has same marshalling rule for both bytom and vapor
	TransactionStatus *bc.TransactionStatus `json:"transaction_status"`
}

func (n *Node) getRawBlock(req *getRawBlockReq) (string, *bc.TransactionStatus, error) {
	url := "/get-raw-block"
	payload, err := json.Marshal(req)
	if err != nil {
		return "", nil, errors.Wrap(err, "json marshal")
	}

	res := &getRawBlockResp{}
	return res.RawBlock, res.TransactionStatus, n.request(url, payload, res)
}

type response struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrDetail string          `json:"error_detail"`
}

func (n *Node) request(path string, payload []byte, respData interface{}) error {
	resp := &response{}
	if err := util.Post(n.hostPort+path, payload, resp); err != nil {
		return err
	}

	if resp.Status != "success" {
		return errors.New(resp.ErrDetail)
	}

	return json.Unmarshal(resp.Data, respData)
}
