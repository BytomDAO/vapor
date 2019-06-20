package chainmgr

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

var (
	maxBlocksPerMsg      = maxHeadersPerMsg + 1
	maxHeadersPerMsg     = 500
	fastSyncPivotGap     = 64
	minGapStartFastSync  = 128
	maxFastSyncBlocksNum = 10000

	errHeadersNum       = errors.New("headers number error")
	errBlocksNum        = errors.New("blocks number error")
	errNoCommonAncestor = errors.New("can't find common ancestor")
)

func (bk *blockKeeper) blockLocator() []*bc.Hash {
	header := bk.chain.BestBlockHeader()
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
			header, err = bk.chain.GetHeaderByHeight(0)
		} else {
			header, err = bk.chain.GetHeaderByHeight(header.Height - step)
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

func (bk *blockKeeper) fastSynchronize() error {
	bk.initFastSyncParameters()

	err := bk.findFastSyncRange()
	if err != nil {
		return err
	}

	if err := bk.fetchSkeleton(); err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Error("failed on fetch skeleton")
		return err
	}

	for i := 0; i < len(bk.skeleton)-1; i++ {
		blocks, err := bk.fetchBlocks(bk.skeleton[i], bk.skeleton[i+1])
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "error": err}).Error("failed on fetch blocks")
			return err
		}

		if err := bk.verifyBlocks(blocks); err != nil {
			log.WithFields(log.Fields{"module": logModule, "error": err}).Error("failed on process blocks")
			return err
		}
	}

	return nil
}

func (bk *blockKeeper) fetchBlocks(startHeader *types.BlockHeader, stopHeader *types.BlockHeader) ([]*types.Block, error) {
	startHash := startHeader.Hash()
	stopHash := stopHeader.Hash()
	bodies, err := bk.requireBlocks(bk.syncPeer.ID(), []*bc.Hash{&startHash}, &stopHash, int(stopHeader.Height-startHeader.Height+1))
	if err != nil {
		return nil, err
	}

	return bodies, nil
}

func (bk *blockKeeper) findCommonAncestor() error {
	headers, err := bk.requireHeaders(bk.syncPeer.ID(), bk.blockLocator(), 1, 0)
	if err != nil {
		return err
	}

	if len(headers) != 1 {
		return errNoCommonAncestor
	}

	bk.commonAncestor = headers[0]
	return nil
}

func (bk *blockKeeper) findFastSyncRange() error {
	if err := bk.findCommonAncestor(); err != nil {
		return err
	}

	gap := bk.syncPeer.Height() - uint64(fastSyncPivotGap) - bk.commonAncestor.Height

	if gap > uint64(maxFastSyncBlocksNum) {
		bk.fastSyncLength = maxFastSyncBlocksNum
		return nil
	}

	bk.fastSyncLength = int(gap)
	return nil
}

func (bk *blockKeeper) initFastSyncParameters() {
	bk.skeleton = make([]*types.BlockHeader, 0)
	bk.commonAncestor = nil
	bk.fastSyncLength = 0
}

func (bk *blockKeeper) locateBlocks(locator []*bc.Hash, stopHash *bc.Hash) ([]*types.Block, error) {
	headers, err := bk.locateHeaders(locator, stopHash, 0, 0)
	if err != nil {
		return nil, err
	}

	blocks := []*types.Block{}
	for i, header := range headers {
		if i >= maxBlocksPerMsg {
			break
		}

		headerHash := header.Hash()
		block, err := bk.chain.GetBlockByHash(&headerHash)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
	}
	return blocks, nil
}

func (bk *blockKeeper) locateHeaders(locator []*bc.Hash, stopHash *bc.Hash, amount int, skip int) ([]*types.BlockHeader, error) {
	startHeader, err := bk.chain.GetHeaderByHeight(0)
	if err != nil {
		return nil, err
	}

	for _, hash := range locator {
		header, err := bk.chain.GetHeaderByHash(hash)
		if err == nil && bk.chain.InMainChain(header.Hash()) {
			startHeader = header
			break
		}
	}

	var stopHeader *types.BlockHeader
	if stopHash != nil {
		stopHeader, err = bk.chain.GetHeaderByHash(stopHash)
	} else {
		stopHeader, err = bk.chain.GetHeaderByHeight(startHeader.Height + uint64((amount-1)*(skip+1)))
	}
	if err != nil {
		return nil, err
	}

	headers := []*types.BlockHeader{}
	for i := startHeader.Height; i <= stopHeader.Height; i += uint64(skip) + 1 {
		header, err := bk.chain.GetHeaderByHeight(i)
		if err != nil {
			return nil, err
		}

		headers = append(headers, header)
	}
	return headers, nil
}

func (bk *blockKeeper) requireBlocks(peerID string, locator []*bc.Hash, stopHash *bc.Hash, length int) ([]*types.Block, error) {
	peer := bk.peers.GetPeer(peerID)
	if peer == nil {
		return nil, errPeerDropped
	}

	if ok := peer.GetBlocks(locator, stopHash); !ok {
		return nil, errPeerDropped
	}

	timeout := time.NewTimer(syncTimeout)
	defer timeout.Stop()

	for {
		select {
		case msg := <-bk.blocksProcessCh:
			if msg.peerID != peerID {
				continue
			}
			if len(msg.blocks) != length {
				return nil, errBlocksNum
			}

			return msg.blocks, nil
		case <-timeout.C:
			return nil, errors.Wrap(errRequestTimeout, "requireBlocks")
		}
	}
}

func (bk *blockKeeper) requireHeaders(peerID string, locator []*bc.Hash, amount int, skip int) ([]*types.BlockHeader, error) {
	peer := bk.peers.GetPeer(peerID)
	if peer == nil {
		return nil, errPeerDropped
	}

	if ok := peer.GetHeaders(locator, amount, skip); !ok {
		return nil, errPeerDropped
	}

	timeout := time.NewTimer(syncTimeout)
	defer timeout.Stop()

	for {
		select {
		case msg := <-bk.headersProcessCh:
			if msg.peerID != peerID {
				continue
			}

			if len(msg.headers) != int(amount) {
				return nil, errHeadersNum
			}

			return msg.headers, nil
		case <-timeout.C:
			return nil, errors.Wrap(errRequestTimeout, "requireHeaders")
		}
	}
}

func (bk *blockKeeper) fetchSkeleton() error {
	startPoint := bk.commonAncestor.Hash()

	bk.skeleton = append(bk.skeleton, bk.commonAncestor)
	if bk.fastSyncLength > maxHeadersPerMsg {
		headers, err := bk.requireHeaders(bk.syncPeer.ID(), []*bc.Hash{&startPoint}, bk.fastSyncLength/maxHeadersPerMsg+1, maxHeadersPerMsg-1)
		if err != nil {
			return err
		}

		bk.skeleton = append(bk.skeleton, headers[1:]...)
	}

	if bk.fastSyncLength%maxHeadersPerMsg != 0 {
		headers, err := bk.requireHeaders(bk.syncPeer.ID(), []*bc.Hash{&startPoint}, 2, bk.fastSyncLength-1)
		if err != nil {
			return err
		}

		bk.skeleton = append(bk.skeleton, headers[1])
	}

	return nil
}

func (bk *blockKeeper) verifyBlocks(blocks []*types.Block) error {
	for _, block := range blocks {
		isOrphan, err := bk.chain.ProcessBlock(block)
		if err != nil {
			return err
		}

		if isOrphan {
			log.WithFields(log.Fields{"module": logModule, "height": block.Height, "hash": block.Hash()}).Warn("fast sync block is orphan")
		}
	}

	return nil
}
