package clients

import (
	"github.com/bytom/bytom/errors"
	"github.com/bytom/vapor/api"
	"github.com/bytom/vapor/protocol/bc/types"
)

type VaporClient struct {
	*apiClient
}

func NewVaporClient(url string) *VaporClient {
	return &VaporClient{newApiClient(url, "")}
}

// GetBlockCount return the latest blockHeight on the chain
func (v *VaporClient) GetBlockCount() (uint64, error) {
	var blockHeight map[string]uint64
	url := v.baseURL + "/get-block-count"
	err := errors.Wrapf(v.request(url, nil, &blockHeight), "GetBlockCount")
	currentBlockHeight := blockHeight["block_count"]
	return currentBlockHeight, err
}

// GetRawBlockByHeight return the Block by BlockHeight
func (v *VaporClient) GetRawBlockByHeight(blockHeight uint64) (*types.Block, error) {
	req := new(api.BlockReq)
	req.BlockHeight = blockHeight
	// GetRawBlock
	url := v.baseURL + "/get-raw-block"
	resp := &api.GetRawBlockResp{}
	err := errors.Wrapf(v.request(url, req, resp), "getRawBlock")
	return resp.RawBlock, err
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
