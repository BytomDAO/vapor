package consensusmgr

import (
	"testing"
	"time"

	"github.com/bytom/vapor/netsync/peers"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
)

type peerMgr struct {
}

func (pm *peerMgr) IsBanned(ip string, level byte, reason string) bool {
	return false
}

func (pm *peerMgr) StopPeerGracefully(string) {
	return
}

type chain struct {
	blocks []uint64
}

func newChain() *chain {
	blocks := make([]uint64, 1, 1)
	blocks[0] = 99
	return &chain{
		blocks: blocks,
	}
}

func (c *chain) BestBlockHeight() uint64 {
	return c.blocks[len(c.blocks)-1]
}

func (c *chain) GetHeaderByHash(*bc.Hash) (*types.BlockHeader, error) {
	return nil, nil
}

func (c *chain) ProcessBlock(block *types.Block) (bool, error) {
	c.blocks = append(c.blocks, block.Height)
	return false, nil
}

func (c *chain) ProcessBlockSignature(signature, pubkey []byte, blockHash *bc.Hash) error {
	return nil
}

func TestBlockFetcher(t *testing.T) {
	peers := peers.NewPeerSet(&peerMgr{})
	testCase := []struct {
		blockMsg *blockMsg
		height   uint64
	}{
		{
			blockMsg: &blockMsg{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height: 100,
					},
				},
			},
			height: 100,
		},
		{
			blockMsg: &blockMsg{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height: 101,
					},
				},
			},
			height: 101,
		},
		{
			blockMsg: &blockMsg{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height: 105,
					},
				},
			},
			height: 101,
		},
		{
			blockMsg: &blockMsg{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height: 200,
					},
				},
			},
			height: 101,
		},
		{
			blockMsg: &blockMsg{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height: 104,
					},
				},
			},
			height: 101,
		},
		{
			blockMsg: &blockMsg{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height: 103,
					},
				},
			},
			height: 101,
		},
		{
			blockMsg: &blockMsg{
				block: &types.Block{
					BlockHeader: types.BlockHeader{
						Height: 102,
					},
				},
			},
			height: 105,
		},
	}
	fetcher := newBlockFetcher(newChain(), peers)

	for i, c := range testCase {
		fetcher.processNewBlock(c.blockMsg)
		time.Sleep(10 * time.Millisecond)
		chainHeight := fetcher.chain.BestBlockHeight()
		if chainHeight != c.height {
			t.Fatalf("test block fetcher error. index %d expected chain height %d but got %d", i, chainHeight, c.height)
		}
	}
}
