package sync

import "github.com/bytom/vapor/protocol/bc/types"

// GetBlockArray return the RawBlockArray by BlockHeight from start to start+length-1
func (b *BlockKeeper) GetBlockArray(start, length uint64) ([]*types.Block, error) {
	blockHeight := start
	data := []*types.Block{}
	for i := uint64(0); i < length; i++ {
		resp, err := b.Node.GetBlockByHeight(blockHeight)
		if err != nil {
			return nil, err
		}

		data = append(data, resp)
		blockHeight++
	}
	return data, nil
}
