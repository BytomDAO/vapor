package netsync

import (
	"errors"
	"reflect"

	log "github.com/sirupsen/logrus"

	cfg "github.com/vapor/config"
	"github.com/vapor/consensus"
	"github.com/vapor/event"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/p2p"
	core "github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	logModule      = "netsync"
	txsMsgMaxTxNum = 1024
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
	StopPeerGracefully(string)
	Start() (bool, error)
	Stop() bool
	IsListening() bool
	DialPeerWithAddress(addr *p2p.NetAddress) error
	Peers() *p2p.PeerSet
}

//SyncManager Sync Manager is responsible for the business layer information synchronization
type SyncManager struct {
	sw           Switch
	genesisHash  bc.Hash
	chain        Chain
	txPool       *core.TxPool
	blockFetcher *blockFetcher
	blockKeeper  *blockKeeper
	peers        *peers.PeerSet

	txSyncCh chan *txSyncMsg
	quitSync chan struct{}
	config   *cfg.Config

	eventDispatcher *event.Dispatcher
	minedBlockSub   *event.Subscription
	txMsgSub        *event.Subscription
}

// NewSyncManager create sync manager and set switch.
func NewSyncManager(config *cfg.Config, chain Chain, txPool *core.TxPool, dispatcher *event.Dispatcher) (*SyncManager, error) {
	sw, err := p2p.NewSwitch(config)
	if err != nil {
		return nil, err
	}

	return newSyncManager(config, sw, chain, txPool, dispatcher)
}

//newSyncManager create a sync manager
func newSyncManager(config *cfg.Config, sw Switch, chain Chain, txPool *core.TxPool, dispatcher *event.Dispatcher) (*SyncManager, error) {
	genesisHeader, err := chain.GetHeaderByHeight(0)
	if err != nil {
		return nil, err
	}
	peers := peers.NewPeerSet(sw)
	manager := &SyncManager{
		sw:              sw,
		genesisHash:     genesisHeader.Hash(),
		txPool:          txPool,
		chain:           chain,
		blockFetcher:    newBlockFetcher(chain, peers),
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

//AddPeer add peer to SyncManager PeerSet
func (sm *SyncManager) AddPeer(peer peers.BasePeer) {
	sm.peers.AddPeer(peer)
}

//BestPeer return the highest p2p peerInfo
func (sm *SyncManager) BestPeer() *peers.PeerInfo {
	peerID, bestHeight := sm.peers.BestPeerInfo(consensus.SFFullNode)
	if peerID == "" || bestHeight == 0 {
		return nil
	}

	return sm.peers.GetPeerInfo(peerID)
}

//DialPeerWithAddress
func (sm *SyncManager) DialPeerWithAddress(addr *p2p.NetAddress) error {
	if sm.config.VaultMode {
		return errVaultModeDialPeer
	}

	return sm.sw.DialPeerWithAddress(addr)
}

//GetNetwork return chain id
func (sm *SyncManager) GetNetwork() string {
	return sm.config.ChainID
}

//GetPeerInfos return peer info of all peers
func (sm *SyncManager) GetPeerInfos() []*peers.PeerInfo {
	return sm.peers.GetPeerInfos()
}

//IsCaughtUp check wheather the peer finish the sync
func (sm *SyncManager) IsCaughtUp() bool {
	bestPeerID, bestPeerHeight := sm.peers.BestPeerInfo(consensus.SFFullNode)
	return bestPeerID == "" || bestPeerHeight <= sm.chain.BestBlockHeight()
}

//RemovePeer del peer from SyncManager PeerSet then disconnect with peer
func (sm *SyncManager) RemovePeer(peer peers.BasePeer) {
	sm.peers.RemovePeer(peer.ID())
}

//StopPeer try to stop peer by given ID
func (sm *SyncManager) StopPeer(peerID string) error {
	sm.peers.RemovePeer(peerID)
	return nil
}

func (sm *SyncManager) handleBlockMsg(peerID string, msg *BlockMessage) {
	block, err := msg.GetBlock()
	if err != nil {
		return
	}
	sm.blockKeeper.processBlock(peerID, block)
}

func (sm *SyncManager) handleBlocksMsg(peerID string, msg *BlocksMessage) {
	blocks, err := msg.GetBlocks()
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Debug("fail on handleBlocksMsg GetBlocks")
		return
	}

	sm.blockKeeper.processBlocks(peerID, blocks)
}

func (sm *SyncManager) handleFilterAddMsg(peerID string, msg *FilterAddMessage) {
	var addresses [][]byte
	addresses = append(addresses, msg.Address)
	sm.peers.AddFilterAddresses(peerID, addresses)

}

func (sm *SyncManager) handleFilterClearMsg(peerID string) {
	sm.peers.ClearFilterAdds(peerID)
}

func (sm *SyncManager) handleFilterLoadMsg(peerID string, msg *FilterLoadMessage) {
	sm.peers.AddFilterAddresses(peerID, msg.Addresses)
}

func (sm *SyncManager) handleGetBlockMsg(peerID string, msg *GetBlockMessage) {
	var block *types.Block
	var err error
	if msg.Height != 0 {
		block, err = sm.chain.GetBlockByHeight(msg.Height)
	} else {
		block, err = sm.chain.GetBlockByHash(msg.GetHash())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleGetBlockMsg get block from chain")
		return
	}
	blockMsg, err := NewBlockMessage(block)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on create block message")
		return
	}
	ok := sm.peers.SendMsg(peerID, BlockchainChannel, blockMsg)
	if !ok {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetBlockMsg sentBlock")
	}
	//todo:mark block
}

func (sm *SyncManager) handleGetBlocksMsg(peerID string, msg *GetBlocksMessage) {
	blocks, err := sm.blockKeeper.locateBlocks(msg.GetBlockLocator(), msg.GetStopHash())
	if err != nil || len(blocks) == 0 {
		return
	}

	totalSize := 0
	sendBlocks := []*types.Block{}
	for _, block := range blocks {
		rawData, err := block.MarshalText()
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("failed on handleGetBlocksMsg marshal block")
			continue
		}

		if totalSize+len(rawData) > MaxBlockchainResponseSize/2 {
			break
		}
		totalSize += len(rawData)
		sendBlocks = append(sendBlocks, block)
	}
	blocksMsg, err := NewBlocksMessage(blocks)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warn("failed on create blocks msg")
		return
	}
	ok := sm.peers.SendMsg(peerID, BlockchainChannel, blocksMsg)
	if !ok {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetBlocksMsg sentBlock")
	}
	//todo: mark block
}

func (sm *SyncManager) handleGetHeadersMsg(peerID string, msg *GetHeadersMessage) {
	headers, err := sm.blockKeeper.locateHeaders(msg.GetBlockLocator(), msg.GetStopHash())
	if err != nil || len(headers) == 0 {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Debug("failed on handleGetHeadersMsg locateHeaders")
		return
	}

	headersMsg, err := NewHeadersMessage(headers)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warn("fail on create headers msg")
		return
	}

	ok := sm.peers.SendMsg(peerID, BlockchainChannel, headersMsg)

	if !ok {
		log.WithFields(log.Fields{"module": logModule}).Error("fail on handleGetHeadersMsg sentBlock")
	}
}

func (sm *SyncManager) handleGetMerkleBlockMsg(peerID string, msg *GetMerkleBlockMessage) {
	var err error
	var block *types.Block
	if msg.Height != 0 {
		block, err = sm.chain.GetBlockByHeight(msg.Height)
	} else {
		block, err = sm.chain.GetBlockByHash(msg.GetHash())
	}
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleGetMerkleBlockMsg get block from chain")
		return
	}

	blockHash := block.Hash()
	txStatus, err := sm.chain.GetTransactionStatus(&blockHash)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleGetMerkleBlockMsg get transaction status")
		return
	}

	merkleBlockMsg := NewMerkleBlockMessage()
	if err := merkleBlockMsg.SetRawBlockHeader(block.BlockHeader); err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleGetMerkleBlockMsg set block header")
		return
	}

	relatedTxs, relatedStatuses := sm.peers.GetRelatedTxAndStatus(peerID, block.Transactions, txStatus)

	txHashes, txFlags := types.GetTxMerkleTreeProof(block.Transactions, relatedTxs)
	if err := merkleBlockMsg.SetTxInfo(txHashes, txFlags, relatedTxs); err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleGetMerkleBlockMsg set tx info")
		return
	}

	statusHashes := types.GetStatusMerkleTreeProof(txStatus.VerifyStatus, txFlags)
	if err := merkleBlockMsg.SetStatusInfo(statusHashes, relatedStatuses); err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleGetMerkleBlockMsg set status info")
		return
	}
	ok := sm.peers.SendMsg(peerID, BlockchainChannel, merkleBlockMsg)
	if !ok {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on handleGetMerkleBlockMsg sentMerkleBlock")
	}
}

func (sm *SyncManager) handleHeadersMsg(peerID string, msg *HeadersMessage) {
	headers, err := msg.GetHeaders()
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Debug("fail on handleHeadersMsg GetHeaders")
		return
	}

	sm.blockKeeper.processHeaders(peerID, headers)
}

func (sm *SyncManager) handleMineBlockMsg(peerID string, msg *MineBlockMessage) {
	block, err := msg.GetMineBlock()
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Warning("fail on handleMineBlockMsg GetMineBlock")
		return
	}

	hash := block.Hash()
	sm.peers.MarkBlock(peerID, &hash)
	sm.blockFetcher.processNewBlock(&blockMsg{peerID: peerID, block: block})
	sm.peers.SetStatus(peerID, block.Height, &hash)
}

func (sm *SyncManager) handleStatusMsg(peerID string, msg *StatusMessage) {
	sm.peers.SetStatus(peerID, msg.Height, msg.GetHash())
}

func (sm *SyncManager) handleTransactionMsg(peerID string, msg *TransactionMessage) {
	tx, err := msg.GetTransaction()
	if err != nil {
		sm.peers.AddBanScore(peerID, 0, 10, "fail on get tx from message")
		return
	}

	if isOrphan, err := sm.chain.ValidateTx(tx); err != nil && err != core.ErrDustTx && !isOrphan {
		sm.peers.AddBanScore(peerID, 10, 0, "fail on validate tx transaction")
	}
}

func (sm *SyncManager) handleTransactionsMsg(peerID string, msg *TransactionsMessage) {
	txs, err := msg.GetTransactions()
	if err != nil {
		sm.peers.AddBanScore(peerID, 0, 20, "fail on get txs from message")
		return
	}

	if len(txs) > txsMsgMaxTxNum {
		sm.peers.AddBanScore(peerID, 20, 0, "exceeded the maximum tx number limit")
		return
	}

	for _, tx := range txs {
		if isOrphan, err := sm.chain.ValidateTx(tx); err != nil && !isOrphan {
			sm.peers.AddBanScore(peerID, 10, 0, "fail on validate tx transaction")
			return
		}
		sm.peers.MarkTx(peerID, &tx.ID)
	}
}

//IsListening
func (sm *SyncManager) IsListening() bool {
	if sm.config.VaultMode {
		return false
	}
	return sm.sw.IsListening()
}

func (sm *SyncManager) PeerCount() int {
	if sm.config.VaultMode {
		return 0
	}
	return len(sm.sw.Peers().List())
}

func (sm *SyncManager) processMsg(peerID string, msgType byte, msg BlockchainMessage) {
	if sm.peers.GetPeer(peerID) == nil {
		return
	}

	log.WithFields(log.Fields{
		"module":  logModule,
		"peer":    peerID,
		"type":    reflect.TypeOf(msg),
		"message": msg.String(),
	}).Info("receive message from peer")

	switch msg := msg.(type) {
	case *GetBlockMessage:
		sm.handleGetBlockMsg(peerID, msg)

	case *BlockMessage:
		sm.handleBlockMsg(peerID, msg)

	case *StatusMessage:
		sm.handleStatusMsg(peerID, msg)

	case *TransactionMessage:
		sm.handleTransactionMsg(peerID, msg)

	case *TransactionsMessage:
		sm.handleTransactionsMsg(peerID, msg)

	case *MineBlockMessage:
		sm.handleMineBlockMsg(peerID, msg)

	case *GetHeadersMessage:
		sm.handleGetHeadersMsg(peerID, msg)

	case *HeadersMessage:
		sm.handleHeadersMsg(peerID, msg)

	case *GetBlocksMessage:
		sm.handleGetBlocksMsg(peerID, msg)

	case *BlocksMessage:
		sm.handleBlocksMsg(peerID, msg)

	case *FilterLoadMessage:
		sm.handleFilterLoadMsg(peerID, msg)

	case *FilterAddMessage:
		sm.handleFilterAddMsg(peerID, msg)

	case *FilterClearMessage:
		sm.handleFilterClearMsg(peerID)

	case *GetMerkleBlockMessage:
		sm.handleGetMerkleBlockMsg(peerID, msg)

	default:
		log.WithFields(log.Fields{
			"module":       logModule,
			"peerID":       peerID,
			"message_type": reflect.TypeOf(msg),
		}).Error("unhandled message type")
	}
}

func (sm *SyncManager) SendStatus(peerID string) bool {
	msg := NewStatusMessage(sm.chain.BestBlockHeader())
	ok := sm.peers.SendMsg(peerID, BlockchainChannel, msg)
	if !ok {
		log.WithFields(log.Fields{"module": logModule}).Error("fail on send status msg")
	}
	//todo: mark status
	return ok
}

func (sm *SyncManager) Start() error {
	var err error
	if _, err = sm.sw.Start(); err != nil {
		log.Error("switch start err")
		return err
	}

	sm.minedBlockSub, err = sm.eventDispatcher.Subscribe(event.NewMinedBlockEvent{})
	if err != nil {
		return err
	}

	sm.txMsgSub, err = sm.eventDispatcher.Subscribe(core.TxMsgEvent{})
	if err != nil {
		return err
	}

	// broadcast transactions
	go sm.txBroadcastLoop()
	go sm.minedBroadcastLoop()
	go sm.txSyncLoop()

	return nil
}

//Stop stop sync manager
func (sm *SyncManager) Stop() {
	close(sm.quitSync)
	sm.minedBlockSub.Unsubscribe()
	if !sm.config.VaultMode {
		sm.sw.Stop()
	}
}

func (sm *SyncManager) minedBroadcastLoop() {
	for {
		select {
		case obj, ok := <-sm.minedBlockSub.Chan():
			if !ok {
				log.WithFields(log.Fields{"module": logModule}).Warning("mined block subscription channel closed")
				return
			}

			ev, ok := obj.Data.(event.NewMinedBlockEvent)
			if !ok {
				log.WithFields(log.Fields{"module": logModule}).Error("event type error")
				continue
			}
			minedMsg, _ := newMinedBlockBroadcastMsg(&ev.Block, BlockchainChannel)
			if err := sm.peers.BroadcastMsg(minedMsg); err != nil {
				//if err := sm.peers.broadcastMinedBlock(&ev.Block); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on broadcast mine block")
				continue
			}

		case <-sm.quitSync:
			return
		}
	}
}
