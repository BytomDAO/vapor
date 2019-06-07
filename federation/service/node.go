package service

import (
	"encoding/json"

	"github.com/vapor/errors"
	"github.com/vapor/federation/util"
	"github.com/vapor/protocol/bc/types"
)

// Node can invoke the api which provide by the full node server
type Node struct {
	ip string
}

// Node create a api client with target server
func NewNode(ip string) *Node {
	return &Node{ip: ip}
}

// TODO: FK
func (n *Node) GetBlockByHash(hash string) (interface{}, interface{}, error) {
	return n.getRawBlock(&getRawBlockReq{BlockHash: hash})
}

// TODO: FK
func (n *Node) GetBlockByHeight(height uint64) (interface{}, interface{}, error) {
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

// TODO: FK
type getRawBlockResp struct {
	RawBlock          interface{} `json:"raw_block"`
	TransactionStatus interface{} `json:"transaction_status"`
}

// TODO: FK
func (n *Node) getRawBlock(req *getRawBlockReq) (interface{}, interface{}, error) {
	url := "/get-raw-block"
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, nil, errors.Wrap(err, "json marshal")
	}

	res := &getRawBlockResp{}
	return res.RawBlock, res.TransactionStatus, n.request(url, payload, res)
}

// TODO: FK
type submitTxReq struct {
	Tx *types.Tx `json:"raw_transaction"`
}

type submitTxResp struct {
	TxID string `json:"tx_id"`
}

func (n *Node) SubmitTx(tx *types.Tx) (string, error) {
	url := "/submit-transaction"
	payload, err := json.Marshal(submitTxReq{Tx: tx})
	if err != nil {
		return "", errors.Wrap(err, "json marshal")
	}

	res := &submitTxResp{}
	return res.TxID, n.request(url, payload, res)
}

type response struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrDetail string          `json:"error_detail"`
}

func (n *Node) request(url string, payload []byte, respData interface{}) error {
	resp := &response{}
	if err := util.Post(n.ip+url, payload, resp); err != nil {
		return err
	}

	if resp.Status != "success" {
		return errors.New(resp.ErrDetail)
	}

	return json.Unmarshal(resp.Data, respData)
}
