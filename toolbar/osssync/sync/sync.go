package sync

import (
	"github.com/bytom/vapor/protocol"
	"github.com/bytom/vapor/protocol/bc/types"
)

// GetLatestDownloadBlockHeight returns the current height of the node wait for download.
func GetLatestDownloadBlockHeight(c *protocol.Chain) uint64 {
	return c.BestBlockHeight()
}

// Sync
func Sync(c *protocol.Chain, blocks []*types.Block) error {
	for i := 0; i < len(blocks); i++ {
		_, err := c.ProcessBlock(blocks[i])
		if err != nil {
			return err
		}
	}
	return nil
}
