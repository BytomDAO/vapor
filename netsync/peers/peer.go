package peers

import (
	"encoding/hex"
	"net"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/flowrate"
	"gopkg.in/fatih/set.v0"

	"github.com/vapor/consensus"
	"github.com/vapor/errors"
	"github.com/vapor/p2p/trust"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	maxKnownTxs           = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownBlocks        = 1024  // Maximum block hashes to keep in the known list (prevent DOS)
	defaultBanThreshold   = uint32(100)
	maxFilterAddressSize  = 50
	maxFilterAddressCount = 1000

	logModule = "peers"
)

var (
	errSendStatusMsg = errors.New("send status msg fail")
	ErrPeerMisbehave = errors.New("peer is misbehave")
)

//BasePeer is the interface for connection level peer
type BasePeer interface {
	Addr() net.Addr
	ID() string
	ServiceFlag() consensus.ServiceFlag
	TrafficStatus() (*flowrate.Status, *flowrate.Status)
	TrySend(byte, interface{}) bool
	IsLAN() bool
}

// PeerInfo indicate peer status snap
type PeerInfo struct {
	ID                  string `json:"peer_id"`
	RemoteAddr          string `json:"remote_addr"`
	Height              uint64 `json:"height"`
	Ping                string `json:"ping"`
	Duration            string `json:"duration"`
	TotalSent           int64  `json:"total_sent"`
	TotalReceived       int64  `json:"total_received"`
	AverageSentRate     int64  `json:"average_sent_rate"`
	AverageReceivedRate int64  `json:"average_received_rate"`
	CurrentSentRate     int64  `json:"current_sent_rate"`
	CurrentReceivedRate int64  `json:"current_received_rate"`
}

type Peer struct {
	BasePeer
	mtx         sync.RWMutex
	services    consensus.ServiceFlag
	height      uint64
	hash        *bc.Hash
	banScore    trust.DynamicBanScore
	knownTxs    *set.Set // Set of transaction hashes known to be known by this peer
	knownBlocks *set.Set // Set of block hashes known to be known by this peer
	knownStatus uint64   // Set of chain status known to be known by this peer
	filterAdds  *set.Set // Set of addresses that the spv node cares about.
}

func newPeer(basePeer BasePeer) *Peer {
	return &Peer{
		BasePeer:    basePeer,
		services:    basePeer.ServiceFlag(),
		knownTxs:    set.New(),
		knownBlocks: set.New(),
		filterAdds:  set.New(),
	}
}

func (p *Peer) bestHeight() uint64 {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.height
}

func (p *Peer) addBanScore(persistent, transient uint32, reason string) bool {
	score := p.banScore.Increase(persistent, transient)
	if score > defaultBanThreshold {
		log.WithFields(log.Fields{
			"module":  logModule,
			"address": p.Addr(),
			"score":   score,
			"reason":  reason,
		}).Errorf("banning and disconnecting")
		return true
	}

	warnThreshold := defaultBanThreshold >> 1
	if score > warnThreshold {
		log.WithFields(log.Fields{
			"module":  logModule,
			"address": p.Addr(),
			"score":   score,
			"reason":  reason,
		}).Warning("ban score increasing")
	}
	return false
}

func (p *Peer) addFilterAddress(address []byte) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	if p.filterAdds.Size() >= maxFilterAddressCount {
		log.WithField("module", logModule).Warn("the count of filter addresses is greater than limit")
		return
	}
	if len(address) > maxFilterAddressSize {
		log.WithField("module", logModule).Warn("the size of filter address is greater than limit")
		return
	}

	p.filterAdds.Add(hex.EncodeToString(address))
}

func (p *Peer) addFilterAddresses(addresses [][]byte) {
	if !p.filterAdds.IsEmpty() {
		p.filterAdds.Clear()
	}
	for _, address := range addresses {
		p.addFilterAddress(address)
	}
}

//func (p *Peer) GetBlockByHeight(height uint64) bool {
//	return p.TrySend(BlockchainChannel, msg)
//}

//func (p *Peer) getBlocks(locator []*bc.Hash, stopHash *bc.Hash) bool {
//	return p.TrySend(BlockchainChannel, msg)
//}

//func (p *Peer) getHeaders(locator []*bc.Hash, stopHash *bc.Hash) bool {
//	msg := struct{ BlockchainMessage }{NewGetHeadersMessage(locator, stopHash)}
//	return p.TrySend(BlockchainChannel, msg)
//}

func (p *Peer) getPeerInfo() *PeerInfo {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	sentStatus, receivedStatus := p.TrafficStatus()
	ping := sentStatus.Idle - receivedStatus.Idle
	if receivedStatus.Idle > sentStatus.Idle {
		ping = -ping
	}

	return &PeerInfo{
		ID:                  p.ID(),
		RemoteAddr:          p.Addr().String(),
		Height:              p.height,
		Ping:                ping.String(),
		Duration:            sentStatus.Duration.String(),
		TotalSent:           sentStatus.Bytes,
		TotalReceived:       receivedStatus.Bytes,
		AverageSentRate:     sentStatus.AvgRate,
		AverageReceivedRate: receivedStatus.AvgRate,
		CurrentSentRate:     sentStatus.CurRate,
		CurrentReceivedRate: receivedStatus.CurRate,
	}
}

func (p *Peer) getRelatedTxAndStatus(txs []*types.Tx, txStatuses *bc.TransactionStatus) ([]*types.Tx, []*bc.TxVerifyResult) {
	var relatedTxs []*types.Tx
	var relatedStatuses []*bc.TxVerifyResult
	for i, tx := range txs {
		if p.isRelatedTx(tx) {
			relatedTxs = append(relatedTxs, tx)
			relatedStatuses = append(relatedStatuses, txStatuses.VerifyStatus[i])
		}
	}
	return relatedTxs, relatedStatuses
}

func (p *Peer) isRelatedTx(tx *types.Tx) bool {
	for _, input := range tx.Inputs {
		switch inp := input.TypedInput.(type) {
		case *types.SpendInput:
			if p.filterAdds.Has(hex.EncodeToString(inp.ControlProgram)) {
				return true
			}
		}
	}
	for _, output := range tx.Outputs {
		if p.filterAdds.Has(hex.EncodeToString(output.ControlProgram())) {
			return true
		}
	}
	return false
}

func (p *Peer) isSPVNode() bool {
	return !p.services.IsEnable(consensus.SFFullNode)
}

func (p *Peer) markBlock(hash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	for p.knownBlocks.Size() >= maxKnownBlocks {
		p.knownBlocks.Pop()
	}
	p.knownBlocks.Add(hash.String())
}

func (p *Peer) markNewStatus(height uint64) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.knownStatus = height
}

func (p *Peer) markTransaction(hash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	for p.knownTxs.Size() >= maxKnownTxs {
		p.knownTxs.Pop()
	}
	p.knownTxs.Add(hash.String())
}

//func (p *Peer) sendStatus(header *types.BlockHeader) error {
//	msg := NewStatusMessage(header)
//	if ok := p.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg}); !ok {
//		return errSendStatusMsg
//	}
//	p.markNewStatus(header.Height)
//	return nil
//}

func (p *Peer) setStatus(height uint64, hash *bc.Hash) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	p.height = height
	p.hash = hash
}
