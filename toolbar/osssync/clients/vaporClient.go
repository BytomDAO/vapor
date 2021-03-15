package clients

import (
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/toolbar/apinode"
)

type VaporClient struct {
	*apinode.Node
}

func NewVaporClient(hostPort string) *VaporClient {
	return &VaporClient{apinode.NewNode(hostPort)}
}

// GetBlockCount return the latest blockHeight on the chain
func (v *VaporClient) GetBlockCount() (uint64, error) {
	return v.Node.GetBlockCount()
}

// GetBlockArray return the RawBlockArray by BlockHeight from start to start+length-1
func (v *VaporClient) GetBlockArray(start, length uint64) ([]*types.Block, error) {
	blockHeight := start
	data := []*types.Block{}
	for i := uint64(0); i < length; i++ {
		resp, err := v.Node.GetBlockByHeight(blockHeight)
		if err != nil {
			return nil, err
		}
		data = append(data, resp)
		blockHeight++
	}
	return data, nil
}
