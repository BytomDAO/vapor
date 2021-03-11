package clients

import (
	"fmt"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/vapor/api"
	"github.com/bytom/vapor/protocol/bc/types"
)

type VaporClient struct {
	*apiClient
}

func NewVaporClient() *VaporClient {
	return &VaporClient{newApiClient("http://localhost:9889", "")}
}

// GetBlock return the Block
func (v *VaporClient) GetBlock(req *api.BlockReq) (*api.GetBlockResp, error) {
	url := v.baseURL + fmt.Sprintf("/get-block")
	resp := &api.GetBlockResp{}
	return resp, errors.Wrapf(v.request(url, req, resp), "GetBlock")
}

// GetBlockByBlockHeight return the Block by BlockHeight
func (v *VaporClient) GetBlockByBlockHeight(blockHeight uint64) (*api.GetBlockResp, error) {
	req := new(api.BlockReq)
	req.BlockHeight = blockHeight
	return v.GetBlock(req)
}

// GetBlockArrayByBlockHeight return the BlockArray
func (v *VaporClient) GetBlockArrayByBlockHeight(start, length uint64) ([]*api.GetBlockResp, error) {
	blockHeight := start
	data := []*api.GetBlockResp{}
	for i := uint64(0); i < length; i++ {
		resp, err := v.GetBlockByBlockHeight(blockHeight)
		if err != nil {
			return nil, err
		}
		data = append(data, resp)
		blockHeight++
	}
	return data, nil
}

func (v *VaporClient) GetBlockCount() (uint64, error) {
	var blockHeight map[string]uint64
	url := v.baseURL + fmt.Sprintf("/get-block-count")
	err := errors.Wrapf(v.request(url, nil, &blockHeight), "GetBlockCount")
	currentBlockHeight := blockHeight["block_count"]
	return currentBlockHeight, err
}

// getRawBlock return the Block
func (v *VaporClient) GetRawBlock(req *api.BlockReq) (*types.Block, error) {
	url := v.baseURL + fmt.Sprintf("/get-raw-block")
	resp := &api.GetRawBlockResp{}
	err := errors.Wrapf(v.request(url, req, resp), "getRawBlock")
	return resp.RawBlock, err
}

// GetRawBlockByHeight return the Block by BlockHeight
func (v *VaporClient) GetRawBlockByHeight(blockHeight uint64) (*types.Block, error) {
	req := new(api.BlockReq)
	req.BlockHeight = blockHeight
	return v.GetRawBlock(req)
}

// GetRawBlockArrayByBlockHeight return the RawBlockArray
func (v *VaporClient) GetRawBlockArrayByBlockHeight(start, length uint64) ([]*types.Block, error) {
	blockHeight := start
	data := []*types.Block{}
	for i := uint64(0); i < length; i++ {
		resp, err := v.GetRawBlockByHeight(blockHeight)
		if err != nil {
			return nil, err
		}
		data = append(data, resp)
		blockHeight++
	}
	return data, nil
}