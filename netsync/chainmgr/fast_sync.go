package chainmgr

import (
	log "github.com/sirupsen/logrus"

	"github.com/vapor/errors"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/p2p/security"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

var (
	maxBlocksPerMsg      = uint64(1000)
	maxHeadersPerMsg     = uint64(1000)
	fastSyncPivotGap     = uint64(64)
	minGapStartFastSync  = uint64(128)
	maxFastSyncBlocksNum = uint64(10000)

	errOrphanBlock = errors.New("fast sync block is orphan")
)

type MsgFetcher interface {
	requireBlock(peerID string, height uint64) (*types.Block, error)
	requireBlocks(peerID string, locator []*bc.Hash, stopHash *bc.Hash) ([]*types.Block, error)
}

type fastSync struct {
	chain      Chain
	msgFetcher MsgFetcher
	peers      *peers.PeerSet
	syncPeer   *peers.Peer
	stopHeader *types.BlockHeader
	length     uint64

	quite chan struct{}
}

func newFastSync(chain Chain, msgFether MsgFetcher, peers *peers.PeerSet) *fastSync {
	return &fastSync{
		chain:      chain,
		msgFetcher: msgFether,
		peers:      peers,
		quite:      make(chan struct{}),
	}
}

func (fs *fastSync) blockLocator() []*bc.Hash {
	header := fs.chain.BestBlockHeader()
	locator := []*bc.Hash{}

	step := uint64(1)
	for header != nil {
		headerHash := header.Hash()
		locator = append(locator, &headerHash)
		if header.Height == 0 {
			break
		}

		var err error
		if header.Height < step {
			header, err = fs.chain.GetHeaderByHeight(0)
		} else {
			header, err = fs.chain.GetHeaderByHeight(header.Height - step)
		}
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("blockKeeper fail on get blockLocator")
			break
		}

		if len(locator) >= 9 {
			step *= 2
		}
	}
	return locator
}

func (fs *fastSync) process() error {
	if err := fs.findFastSyncRange(); err != nil {
		return err
	}

	stopHash := fs.stopHeader.Hash()
	for fs.chain.BestBlockHeight() < fs.stopHeader.Height {
		blocks, err := fs.msgFetcher.requireBlocks(fs.syncPeer.ID(), fs.blockLocator(), &stopHash)
		if err != nil {
			fs.peers.ErrorHandler(fs.syncPeer.ID(), security.LevelConnException, err)
			return err
		}

		if err := fs.verifyBlocks(blocks); err != nil {
			fs.peers.ErrorHandler(fs.syncPeer.ID(), security.LevelMsgIllegal, err)
			return err
		}
	}

	log.WithFields(log.Fields{"module": logModule, "height": fs.chain.BestBlockHeight()}).Info("fast sync success")
	return nil
}

func (fs *fastSync) findFastSyncRange() error {
	bestHeight := fs.chain.BestBlockHeight()
	fs.length = fs.syncPeer.IrreversibleHeight() - fastSyncPivotGap - bestHeight
	if fs.length > maxFastSyncBlocksNum {
		fs.length = maxFastSyncBlocksNum
	}

	stopBlock, err := fs.msgFetcher.requireBlock(fs.syncPeer.ID(), bestHeight+fs.length)
	if err != nil {
		return err
	}

	fs.stopHeader = &stopBlock.BlockHeader
	return nil
}

func (fs *fastSync) locateBlocks(locator []*bc.Hash, stopHash *bc.Hash) ([]*types.Block, error) {
	headers, err := fs.locateHeaders(locator, stopHash, 0, maxBlocksPerMsg)
	if err != nil {
		return nil, err
	}

	blocks := []*types.Block{}
	for _, header := range headers {
		headerHash := header.Hash()
		block, err := fs.chain.GetBlockByHash(&headerHash)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
	}
	return blocks, nil
}

func (fs *fastSync) locateHeaders(locator []*bc.Hash, stopHash *bc.Hash, skip uint64, maxNum uint64) ([]*types.BlockHeader, error) {
	startHeader, err := fs.chain.GetHeaderByHeight(0)
	if err != nil {
		return nil, err
	}

	for _, hash := range locator {
		header, err := fs.chain.GetHeaderByHash(hash)
		if err == nil && fs.chain.InMainChain(header.Hash()) {
			startHeader = header
			break
		}
	}

	headers := make([]*types.BlockHeader, 0)
	stopHeader, err := fs.chain.GetHeaderByHash(stopHash)
	if err != nil {
		return headers, nil
	}

	if !fs.chain.InMainChain(*stopHash) {
		return headers, nil
	}

	num := uint64(0)
	for i := startHeader.Height; i <= stopHeader.Height && num < maxNum; i += skip + 1 {
		header, err := fs.chain.GetHeaderByHeight(i)
		if err != nil {
			return nil, err
		}

		headers = append(headers, header)
		num++
	}

	return headers, nil
}

func (fs *fastSync) setSyncPeer(peer *peers.Peer) {
	fs.syncPeer = peer
}

func (fs *fastSync) verifyBlocks(blocks []*types.Block) error {
	for _, block := range blocks {
		isOrphan, err := fs.chain.ProcessBlock(block)
		if err != nil {
			return err
		}

		if isOrphan {
			log.WithFields(log.Fields{"module": logModule, "height": block.Height, "hash": block.Hash()}).Error("fast sync block is orphan")
			return errOrphanBlock
		}
	}

	return nil
}
