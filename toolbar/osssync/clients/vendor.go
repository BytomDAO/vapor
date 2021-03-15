package clients

import (
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/toolbar/apinode"
)

// GetBlockArray return the RawBlockArray by BlockHeight from start to start+length-1
func GetBlockArray(vaporClient *apinode.Node, start, length uint64) ([]*types.Block, error) {
	blockHeight := start
	data := []*types.Block{}
	for i := uint64(0); i < length; i++ {
		resp, err := vaporClient.GetBlockByHeight(blockHeight)
		if err != nil {
			return nil, err
		}
		data = append(data, resp)
		blockHeight++
	}
	return data, nil
}
