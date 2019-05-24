package netsync

import (
	"errors"
	"reflect"

	log "github.com/sirupsen/logrus"

	cfg "github.com/vapor/config"
	"github.com/vapor/consensus"
	"github.com/vapor/event"
	msgs "github.com/vapor/netsync/messages"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/p2p"
	core "github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	logModule = "netsync"
)

var (
	errVaultModeDialPeer = errors.New("can't dial peer in vault mode")
)

// Chain is the interface for Bytom core
type Chain interface {
	BestBlockHeader() *types.BlockHeader
	BestBlockHeight() uint64
	GetBlockByHash(*bc.Hash) (*types.Block, error)
	GetBlockByHeight(uint64) (*types.Block, error)
	GetHeaderByHash(*bc.Hash) (*types.BlockHeader, error)
	GetHeaderByHeight(uint64) (*types.BlockHeader, error)
	GetTransactionStatus(*bc.Hash) (*bc.TransactionStatus, error)
	InMainChain(bc.Hash) bool
	ProcessBlock(*types.Block) (bool, error)
	ValidateTx(*types.Tx) (bool, error)
}

type Switch interface {
	AddReactor(name string, reactor p2p.Reactor) p2p.Reactor
	AddBannedPeer(string) error
	Start() (bool, error)
	Stop() bool
	IsListening() bool
	DialPeerWithAddress(addr *p2p.NetAddress) error
	Peers() *p2p.PeerSet
}

//ChainManager is responsible for the business layer information synchronization
type ChainManager struct {
	sw          Switch
	chain       Chain
	txPool      *core.TxPool
	blockKeeper *blockKeeper
	peers       *peers.PeerSet

	txSyncCh chan *txSyncMsg
	quitSync chan struct{}
	config   *cfg.Config

	eventDispatcher *event.Dispatcher
	txMsgSub        *event.Subscription
}

//NewChainManager create a chain sync manager.
func NewChainManager(config *cfg.Config, sw Switch, chain Chain, txPool *core.TxPool, dispatcher *event.Dispatcher, peers *peers.PeerSet) (*ChainManager, error) {
	manager := &ChainManager{
		sw:              sw,
		txPool:          txPool,
		chain:           chain,
		blockKeeper:     newBlockKeeper(chain, peers),
		peers:           peers,
		txSyncCh:        make(chan *txSyncMsg),
		quitSync:        make(chan struct{}),
		config:          config,
		eventDispatcher: dispatcher,
	}

	if !config.VaultMode {
		protocolReactor := NewProtocolReactor(manager)
		manager.sw.AddReactor("PROTOCOL", protocolReactor)
	}
	return manager, nil
}

func (cm *ChainManager) AddPeer(peer peers.BasePeer) {
	cm.peers.AddPeer(peer)
}

//IsCaughtUp check wheather the peer finish the sync
func (cm *ChainManager) IsCaughtUp() bool {
	peer := cm.peers.BestPeer(consensus.SFFullNode)
	return peer == nil || peer.Height() <= cm.chain.BestBlockHeight()
}

func (cm *ChainManager) handleBlockMsg(peer *peers.Peer, msg *msgs.BlockMessage) {
	block, err := msg.GetBlock()
	if err != nil {
		return
	}
	cm.blockKeeper.processBlock(peer.ID(), block)
}

func (cm *ChainManager) handleBlocksMsg(peer *peers.Peer, msg *msgs.BlocksMessage) {
	blocks, err := msg.GetBlocks()
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Debug("fail on handleBlocksMsg GetBlocks")
		return
	}

	cm.blockKeeper.processBlocks(peer.ID(), blocks)
}

func (cm *ChainManager) handleFilterAddMsg(peer *peers.Peer, msg *msgs.FilterAddMessage) {
	peer.AddFilterAddress(msg.Address)
}

func (cm *ChainManager) handleFilterClearMsg(peer *peers.Peer) {
	peer.FilterClear()
}

func (cm *ChainManager) handleFilterLoadMsg(peer *peers.Peer, msg *msgs.FilterLoadMessage) {
	peer.AddFilterAddresses(msg.Addresses)
}

func (cm *ChainManager) handleGetBlockMsg(peer *peers.Peer, msg *msgs.GetBlockMessage) {
	var block *types.Block
	var err error
	if msg.Height != 0 {
		block, err = cm.chain.GetBlockByHeight(msg.Height)
	} else {
		block, err = cm.chain.GetBlockByHash(msg.GetHash())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleGetBlockMsg get block from chain")
		return
	}

	ok, err := peer.SendBlock(block)
	if !ok {
		cm.peers.RemovePeer(peer.ID())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetBlockMsg sentBlock")
	}
}

func (cm *ChainManager) handleGetBlocksMsg(peer *peers.Peer, msg *msgs.GetBlocksMessage) {
	blocks, err := cm.blockKeeper.locateBlocks(msg.GetBlockLocator(), msg.GetStopHash())
	if err != nil || len(blocks) == 0 {
		return
	}

	totalSize := 0
	sendBlocks := []*types.Block{}
	for _, block := range blocks {
		rawData, err := block.MarshalText()
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetBlocksMsg marshal block")
			continue
		}

		if totalSize+len(rawData) > msgs.MaxBlockchainResponseSize/2 {
			break
		}
		totalSize += len(rawData)
		sendBlocks = append(sendBlocks, block)
	}

	ok, err := peer.SendBlocks(sendBlocks)
	if !ok {
		cm.peers.RemovePeer(peer.ID())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetBlocksMsg sentBlock")
	}
}

func (cm *ChainManager) handleGetHeadersMsg(peer *peers.Peer, msg *msgs.GetHeadersMessage) {
	headers, err := cm.blockKeeper.locateHeaders(msg.GetBlockLocator(), msg.GetStopHash())
	if err != nil || len(headers) == 0 {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Debug("fail on handleGetHeadersMsg locateHeaders")
		return
	}

	ok, err := peer.SendHeaders(headers)
	if !ok {
		cm.peers.RemovePeer(peer.ID())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetHeadersMsg sentBlock")
	}
}

func (cm *ChainManager) handleGetMerkleBlockMsg(peer *peers.Peer, msg *msgs.GetMerkleBlockMessage) {
	var err error
	var block *types.Block
	if msg.Height != 0 {
		block, err = cm.chain.GetBlockByHeight(msg.Height)
	} else {
		block, err = cm.chain.GetBlockByHash(msg.GetHash())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleGetMerkleBlockMsg get block from chain")
		return
	}

	blockHash := block.Hash()
	txStatus, err := cm.chain.GetTransactionStatus(&blockHash)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleGetMerkleBlockMsg get transaction status")
		return
	}

	ok, err := peer.SendMerkleBlock(block, txStatus)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetMerkleBlockMsg sentMerkleBlock")
		return
	}

	if !ok {
		cm.peers.RemovePeer(peer.ID())
	}
}

func (cm *ChainManager) handleHeadersMsg(peer *peers.Peer, msg *msgs.HeadersMessage) {
	headers, err := msg.GetHeaders()
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Debug("fail on handleHeadersMsg GetHeaders")
		return
	}

	cm.blockKeeper.processHeaders(peer.ID(), headers)
}

func (cm *ChainManager) handleStatusMsg(basePeer peers.BasePeer, msg *msgs.StatusMessage) {
	if peer := cm.peers.GetPeer(basePeer.ID()); peer != nil {
		peer.SetStatus(msg.Height, msg.GetHash())
		return
	}
}

func (cm *ChainManager) handleTransactionMsg(peer *peers.Peer, msg *msgs.TransactionMessage) {
	tx, err := msg.GetTransaction()
	if err != nil {
		cm.peers.AddBanScore(peer.ID(), 0, 10, "fail on get tx from message")
		return
	}

	if isOrphan, err := cm.chain.ValidateTx(tx); err != nil && err != core.ErrDustTx && !isOrphan {
		cm.peers.AddBanScore(peer.ID(), 10, 0, "fail on validate tx transaction")
	}
	cm.peers.MarkTx(peer.ID(), tx.ID)
}

func (cm *ChainManager) handleTransactionsMsg(peer *peers.Peer, msg *msgs.TransactionsMessage) {
	txs, err := msg.GetTransactions()
	if err != nil {
		cm.peers.AddBanScore(peer.ID(), 0, 20, "fail on get txs from message")
		return
	}

	if len(txs) > msgs.TxsMsgMaxTxNum {
		cm.peers.AddBanScore(peer.ID(), 20, 0, "exceeded the maximum tx number limit")
		return
	}

	for _, tx := range txs {
		if isOrphan, err := cm.chain.ValidateTx(tx); err != nil && !isOrphan {
			cm.peers.AddBanScore(peer.ID(), 10, 0, "fail on validate tx transaction")
			return
		}
		cm.peers.MarkTx(peer.ID(), tx.ID)
	}
}

func (cm *ChainManager) processMsg(basePeer peers.BasePeer, msgType byte, msg msgs.BlockchainMessage) {
	peer := cm.peers.GetPeer(basePeer.ID())
	if peer == nil {
		return
	}

	log.WithFields(log.Fields{
		"module":  logModule,
		"peer":    basePeer.Addr(),
		"type":    reflect.TypeOf(msg),
		"message": msg.String(),
	}).Info("receive message from peer")

	switch msg := msg.(type) {
	case *msgs.GetBlockMessage:
		cm.handleGetBlockMsg(peer, msg)

	case *msgs.BlockMessage:
		cm.handleBlockMsg(peer, msg)

	case *msgs.StatusMessage:
		cm.handleStatusMsg(basePeer, msg)

	case *msgs.TransactionMessage:
		cm.handleTransactionMsg(peer, msg)

	case *msgs.TransactionsMessage:
		cm.handleTransactionsMsg(peer, msg)

	case *msgs.GetHeadersMessage:
		cm.handleGetHeadersMsg(peer, msg)

	case *msgs.HeadersMessage:
		cm.handleHeadersMsg(peer, msg)

	case *msgs.GetBlocksMessage:
		cm.handleGetBlocksMsg(peer, msg)

	case *msgs.BlocksMessage:
		cm.handleBlocksMsg(peer, msg)

	case *msgs.FilterLoadMessage:
		cm.handleFilterLoadMsg(peer, msg)

	case *msgs.FilterAddMessage:
		cm.handleFilterAddMsg(peer, msg)

	case *msgs.FilterClearMessage:
		cm.handleFilterClearMsg(peer)

	case *msgs.GetMerkleBlockMessage:
		cm.handleGetMerkleBlockMsg(peer, msg)

	default:
		log.WithFields(log.Fields{
			"module":       logModule,
			"peer":         basePeer.Addr(),
			"message_type": reflect.TypeOf(msg),
		}).Error("unhandled message type")
	}
}

func (cm *ChainManager) RemovePeer(peerID string) {
	cm.peers.RemovePeer(peerID)
}

func (cm *ChainManager) SendStatus(peer peers.BasePeer) error {
	p := cm.peers.GetPeer(peer.ID())
	if p == nil {
		return errors.New("invalid peer")
	}

	if err := p.SendStatus(cm.chain.BestBlockHeader()); err != nil {
		cm.peers.RemovePeer(p.ID())
		return err
	}
	return nil
}

func (cm *ChainManager) Start() error {
	var err error
	cm.txMsgSub, err = cm.eventDispatcher.Subscribe(core.TxMsgEvent{})
	if err != nil {
		return err
	}

	// broadcast transactions
	go cm.txBroadcastLoop()
	go cm.txSyncLoop()

	return nil
}

//Stop stop sync manager
func (cm *ChainManager) Stop() {
	close(cm.quitSync)
}
