package peers

import (
	"reflect"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

//防止重传
//MarkBlock(peerID string, hash *bc.Hash)
//MarkTx(peerID string, hash *bc.Hash)
//peersWithoutBlock(hash *bc.Hash) []*peer
//peersWithoutNewStatus(height uint64) []*peer
//peersWithoutTx(hash *bc.Hash) []*peer

//过滤有效交易
//AddFilterAddresses(peerID string, addresses [][]byte)
//ClearFilterAdds(peerID string)
//FilterValidTxs(peerID string, txs []*types.Tx)
//GetRelatedTxAndStatus(peerID string, txs []*types.Tx, txStatuses *bc.TransactionStatus)

//传输
//BroadcastMinedBlock(block *types.Block)
//BroadcastNewStatus(bestBlock *types.Block) error
//BroadcastTx(tx *types.Tx) error
//SendMsg(peerID string, msgChannel byte, msg interface{})

func (ps *PeerSet) AddFilterAddresses(peerID string, addresses [][]byte) {
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return
	}
	peer.addFilterAddresses(addresses)
}

type BroadcastMsg interface {
	Filter(ps *PeerSet) []string
	Mark(ps *PeerSet, peers []string)
	GetChan() byte
	GetMsg() interface{}
	MsgString() string
}

func (ps *PeerSet) BroadcastMsg(bm BroadcastMsg) error {
	peers := bm.Filter(ps)
	peersSuccess := make([]string, 0)
	for _, peer := range peers {
		if ok := ps.SendMsg(peer, bm.GetChan(), bm.GetMsg()); !ok {
			log.WithFields(log.Fields{"module": logModule, "peer": peer, "type": reflect.TypeOf(bm.GetMsg()), "message": bm.MsgString()}).Warning("send message to peer error")
			continue
		}
		peersSuccess = append(peersSuccess, peer)
	}
	bm.Mark(ps, peersSuccess)
	return nil
}

//func (ps *PeerSet) BroadcastMinedBlock(block *types.Block) error {
//	msg, err := NewMinedBlockMessage(block)
//	if err != nil {
//		return errors.Wrap(err, "fail on broadcast mined block")
//	}
//
//	hash := block.Hash()
//	peers := ps.peersWithoutBlock(&hash)
//	for _, peer := range peers {
//		if peer.isSPVNode() {
//			continue
//		}
//
//		if ok := ps.SendMsg(peer.ID(), BlockchainChannel, msg); !ok {
//			log.WithFields(log.Fields{"module": logModule, "peer": peer.Addr(), "type": reflect.TypeOf(msg), "message": msg.String()}).Warning("send message to peer error")
//			continue
//		}
//
//		peer.markBlock(&hash)
//		peer.markNewStatus(block.Height)
//	}
//	return nil
//}

//func (ps *PeerSet) BroadcastNewStatus(bestBlock *types.Block) error {
//	msg := NewStatusMessage(&bestBlock.BlockHeader)
//	peers := ps.peersWithoutNewStatus(bestBlock.Height)
//	for _, peer := range peers {
//		if ok := ps.SendMsg(peer.ID(), BlockchainChannel, msg); !ok {
//			log.WithFields(log.Fields{"module": logModule, "peer": peer.Addr(), "type": reflect.TypeOf(msg), "message": msg.String()}).Warning("send message to peer error")
//			continue
//		}
//		peer.markNewStatus(bestBlock.Height)
//	}
//	return nil
//}

//func (ps *PeerSet) BroadcastTx(tx *types.Tx) error {
//	msg, err := NewTransactionMessage(tx)
//	if err != nil {
//		return errors.Wrap(err, "fail on broadcast tx")
//	}
//
//	peers := ps.peersWithoutTx(&tx.ID)
//	for _, peer := range peers {
//		if peer.isSPVNode() && !peer.isRelatedTx(tx) {
//			continue
//		}
//		if ok := ps.SendMsg(peer.ID(), BlockchainChannel, msg); !ok {
//			log.WithFields(log.Fields{"module": logModule, "peer": peer.Addr(), "type": reflect.TypeOf(msg), "message": msg.String()}).Warning("send message to peer error")
//			continue
//		}
//		peer.markTransaction(&tx.ID)
//	}
//	return nil
//}

func (ps *PeerSet) ClearFilterAdds(peerID string) {
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return
	}
	peer.filterAdds.Clear()
}

func (ps *PeerSet) FilterValidTxs(peerID string, txs []*types.Tx) []*types.Tx {
	validTxs := make([]*types.Tx, 0, len(txs))
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return validTxs
	}

	for _, tx := range txs {
		if peer.isSPVNode() && !peer.isRelatedTx(tx) || peer.knownTxs.Has(tx.ID.String()) {
			continue
		}

		validTxs = append(validTxs, tx)
	}
	return validTxs
}

func (ps *PeerSet) GetRelatedTxAndStatus(peerID string, txs []*types.Tx, txStatuses *bc.TransactionStatus) ([]*types.Tx, []*bc.TxVerifyResult) {
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return nil, nil
	}
	return peer.getRelatedTxAndStatus(txs, txStatuses)
}

func (ps *PeerSet) MarkBlock(peerID string, hash *bc.Hash) {
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return
	}
	peer.markBlock(hash)
}

func (ps *PeerSet) MarkStatus(peerID string, height uint64) {
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return
	}
	peer.markNewStatus(height)
}

func (ps *PeerSet) MarkTx(peerID string, hash *bc.Hash) {
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return
	}
	peer.markTransaction(hash)
}

func (ps *PeerSet) PeersWithoutBlock(hash bc.Hash) []string {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	var peers []string
	for _, peer := range ps.peers {
		if !peer.knownBlocks.Has(hash.String()) {
			peers = append(peers, peer.ID())
		}
	}
	return peers
}

func (ps *PeerSet) PeersWithoutNewStatus(height uint64) []string {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	var peers []string
	for _, peer := range ps.peers {
		if peer.knownStatus < height {
			peers = append(peers, peer.ID())
		}
	}
	return peers
}

func (ps *PeerSet) PeersWithoutTx(hash bc.Hash) []string {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	var peers []string
	for _, peer := range ps.peers {
		if !peer.knownTxs.Has(hash.String()) {
			peers = append(peers, peer.ID())
		}
	}
	return peers
}

func (ps *PeerSet) SendMsg(peerID string, msgChannel byte, msg interface{}) bool {
	peer := ps.GetPeer(peerID)
	if peer == nil {
		return false
	}

	ok := peer.TrySend(msgChannel, msg)
	if !ok {
		ps.RemovePeer(peerID)
	}
	return ok
}
