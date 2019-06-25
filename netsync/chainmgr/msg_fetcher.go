package chainmgr

import (
	"time"

	"github.com/vapor/errors"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	blockProcessChSize   = 1024
	blocksProcessChSize  = 128
	headersProcessChSize = 1024
)

type msgFetcher struct {
	peers *peers.PeerSet

	blockProcessCh   chan *blockMsg
	blocksProcessCh  chan *blocksMsg
	headersProcessCh chan *headersMsg
}

func newMsgFetcher(peers *peers.PeerSet) *msgFetcher {
	return &msgFetcher{
		peers:            peers,
		blockProcessCh:   make(chan *blockMsg, blockProcessChSize),
		blocksProcessCh:  make(chan *blocksMsg, blocksProcessChSize),
		headersProcessCh: make(chan *headersMsg, headersProcessChSize),
	}
}

func (mf *msgFetcher) processBlock(peerID string, block *types.Block) {
	mf.blockProcessCh <- &blockMsg{block: block, peerID: peerID}
}

func (mf *msgFetcher) processBlocks(peerID string, blocks []*types.Block) {
	mf.blocksProcessCh <- &blocksMsg{blocks: blocks, peerID: peerID}
}

func (mf *msgFetcher) processHeaders(peerID string, headers []*types.BlockHeader) {
	mf.headersProcessCh <- &headersMsg{headers: headers, peerID: peerID}
}

func (mf *msgFetcher) requireBlock(peerID string, height uint64) (*types.Block, error) {
	peer := mf.peers.GetPeer(peerID)
	if peer == nil {
		return nil, errPeerDropped
	}

	if ok := peer.GetBlockByHeight(height); !ok {
		return nil, errPeerDropped
	}

	timeout := time.NewTimer(syncTimeout)
	defer timeout.Stop()

	for {
		select {
		case msg := <-mf.blockProcessCh:
			if msg.peerID != peerID {
				continue
			}
			if msg.block.Height != height {
				continue
			}
			return msg.block, nil
		case <-timeout.C:
			return nil, errors.Wrap(errRequestTimeout, "requireBlock")
		}
	}
}

func (mf *msgFetcher) requireBlocks(peerID string, locator []*bc.Hash, stopHash *bc.Hash) ([]*types.Block, error) {
	peer := mf.peers.GetPeer(peerID)
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
		case msg := <-mf.blocksProcessCh:
			if msg.peerID != peerID {
				continue
			}

			return msg.blocks, nil
		case <-timeout.C:
			return nil, errors.Wrap(errRequestTimeout, "requireBlocks")
		}
	}
}

func (mf *msgFetcher) requireHeaders(peerID string, locator []*bc.Hash, stopHash *bc.Hash, skip uint64) ([]*types.BlockHeader, error) {
	peer := mf.peers.GetPeer(peerID)
	if peer == nil {
		return nil, errPeerDropped
	}

	if ok := peer.GetHeaders(locator, stopHash, skip); !ok {
		return nil, errPeerDropped
	}

	timeout := time.NewTimer(syncTimeout)
	defer timeout.Stop()

	for {
		select {
		case msg := <-mf.headersProcessCh:
			if msg.peerID != peerID {
				continue
			}

			return msg.headers, nil
		case <-timeout.C:
			return nil, errors.Wrap(errRequestTimeout, "requireHeaders")
		}
	}
}
