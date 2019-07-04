package wallet

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/account"
	"github.com/vapor/asset"
	"github.com/vapor/blockchain/pseudohsm"
	"github.com/vapor/errors"
	"github.com/vapor/event"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	//SINGLE single sign
	SINGLE    = 1
	logModule = "wallet"
)

var (
	currentVersion = uint(1)

	errBestBlockNotFoundInCore = errors.New("best block not found in core")
	errWalletVersionMismatch   = errors.New("wallet version mismatch")
	ErrGetWalletStatusInfo     = errors.New("failed get wallet info")
)

//StatusInfo is base valid block info to handle orphan block rollback
type StatusInfo struct {
	Version    uint
	WorkHeight uint64
	WorkHash   bc.Hash
	BestHeight uint64
	BestHash   bc.Hash
}

//Wallet is related to storing account unspent outputs
type Wallet struct {
	store           WalletStore
	rw              sync.RWMutex
	Status          StatusInfo
	TxIndexFlag     bool
	AccountMgr      *account.Manager
	AssetReg        *asset.Registry
	Hsm             *pseudohsm.HSM
	chain           *protocol.Chain
	RecoveryMgr     *recoveryManager
	EventDispatcher *event.Dispatcher
	txMsgSub        *event.Subscription

	rescanCh chan struct{}
}

//NewWallet return a new wallet instance
func NewWallet(store WalletStore, account *account.Manager, asset *asset.Registry, hsm *pseudohsm.HSM, chain *protocol.Chain, dispatcher *event.Dispatcher, txIndexFlag bool) (*Wallet, error) {
	w := &Wallet{
		store:           store,
		AccountMgr:      account,
		AssetReg:        asset,
		chain:           chain,
		Hsm:             hsm,
		RecoveryMgr:     newRecoveryManager(store, account),
		EventDispatcher: dispatcher,
		rescanCh:        make(chan struct{}, 1),
		TxIndexFlag:     txIndexFlag,
	}

	if err := w.LoadWalletInfo(); err != nil {
		return nil, err
	}

	if err := w.RecoveryMgr.LoadStatusInfo(); err != nil {
		return nil, err
	}

	var err error
	w.txMsgSub, err = w.EventDispatcher.Subscribe(protocol.TxMsgEvent{})
	if err != nil {
		return nil, err
	}

	go w.walletUpdater()
	go w.delUnconfirmedTx()
	go w.MemPoolTxQueryLoop()
	return w, nil
}

// MemPoolTxQueryLoop constantly pass a transaction accepted by mempool to the wallet.
func (w *Wallet) MemPoolTxQueryLoop() {
	for {
		select {
		case obj, ok := <-w.txMsgSub.Chan():
			if !ok {
				log.WithFields(log.Fields{"module": logModule}).Warning("tx pool tx msg subscription channel closed")
				return
			}

			ev, ok := obj.Data.(protocol.TxMsgEvent)
			if !ok {
				log.WithFields(log.Fields{"module": logModule}).Error("event type error")
				continue
			}

			switch ev.TxMsg.MsgType {
			case protocol.MsgNewTx:
				w.AddUnconfirmedTx(ev.TxMsg.TxDesc)
			case protocol.MsgRemoveTx:
				w.RemoveUnconfirmedTx(ev.TxMsg.TxDesc)
			default:
				log.WithFields(log.Fields{"module": logModule}).Warn("got unknow message type from the txPool channel")
			}
		}
	}
}

func (w *Wallet) checkWalletInfo() error {
	if w.Status.Version != currentVersion {
		return errWalletVersionMismatch
	} else if !w.chain.BlockExist(&w.Status.BestHash) {
		return errBestBlockNotFoundInCore
	}

	return nil
}

//LoadWalletInfo return stored wallet info and nil,
//if error, return initial wallet info and err
func (w *Wallet) LoadWalletInfo() error {
	walletStatus, err := w.store.GetWalletInfo()
	if walletStatus == nil && err != ErrGetWalletStatusInfo {
		return err
	}
	if walletStatus != nil {
		w.Status = *walletStatus
		err = w.checkWalletInfo()
		if err == nil {
			return nil
		}
		log.WithFields(log.Fields{"module": logModule}).Warn(err.Error())
		w.store.DeleteWalletTransactions()
		w.store.DeleteWalletUTXOs()
	}

	w.Status.Version = currentVersion
	w.Status.WorkHash = bc.Hash{}
	block, err := w.chain.GetBlockByHeight(0)
	if err != nil {
		return err
	}
	return w.AttachBlock(block)
}

func (w *Wallet) commitWalletInfo() error {
	if err := w.store.SetWalletInfo(&w.Status); err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("save wallet info")
		return err
	}
	return nil
}

// AttachBlock attach a new block
func (w *Wallet) AttachBlock(block *types.Block) error {
	w.rw.Lock()
	defer w.rw.Unlock()

	if block.PreviousBlockHash != w.Status.WorkHash {
		log.Warn("wallet skip attachBlock due to status hash not equal to previous hash")
		return nil
	}

	blockHash := block.Hash()
	txStatus, err := w.chain.GetTransactionStatus(&blockHash)
	if err != nil {
		return err
	}

	if err := w.RecoveryMgr.FilterRecoveryTxs(block); err != nil {
		log.WithField("err", err).Error("filter recovery txs")
		w.RecoveryMgr.finished()
	}

	annotatedTxs := w.filterAccountTxs(block, txStatus)
	if err := saveExternalAssetDefinition(block, w.store); err != nil {
		return err
	}
	w.annotateTxsAccount(annotatedTxs)

	if err := w.store.InitBatch(); err != nil {
		return err
	}

	if err := w.indexTransactions(block, txStatus, annotatedTxs); err != nil {
		return err
	}

	w.attachUtxos(block, txStatus)
	w.Status.WorkHeight = block.Height
	w.Status.WorkHash = block.Hash()
	if w.Status.WorkHeight >= w.Status.BestHeight {
		w.Status.BestHeight = w.Status.WorkHeight
		w.Status.BestHash = w.Status.WorkHash
	}

	if err := w.commitWalletInfo(); err != nil {
		return err
	}

	if err := w.store.CommitBatch(); err != nil {
		return err
	}

	return nil
}

// DetachBlock detach a block and rollback state
func (w *Wallet) DetachBlock(block *types.Block) error {
	w.rw.Lock()
	defer w.rw.Unlock()

	blockHash := block.Hash()
	txStatus, err := w.chain.GetTransactionStatus(&blockHash)
	if err != nil {
		return err
	}

	if err := w.store.InitBatch(); err != nil {
		return err
	}

	w.detachUtxos(block, txStatus)
	w.store.DeleteTransactions(w.Status.BestHeight)

	w.Status.BestHeight = block.Height - 1
	w.Status.BestHash = block.PreviousBlockHash

	if w.Status.WorkHeight > w.Status.BestHeight {
		w.Status.WorkHeight = w.Status.BestHeight
		w.Status.WorkHash = w.Status.BestHash
	}
	if err := w.commitWalletInfo(); err != nil {
		return err
	}

	if err := w.store.CommitBatch(); err != nil {
		return err
	}

	return nil
}

//WalletUpdate process every valid block and reverse every invalid block which need to rollback
func (w *Wallet) walletUpdater() {
	for {
		w.getRescanNotification()
		for !w.chain.InMainChain(w.Status.BestHash) {
			block, err := w.chain.GetBlockByHash(&w.Status.BestHash)
			if err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("walletUpdater GetBlockByHash")
				return
			}

			if err := w.DetachBlock(block); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("walletUpdater detachBlock stop")
				return
			}
		}

		block, _ := w.chain.GetBlockByHeight(w.Status.WorkHeight + 1)
		if block == nil {
			w.walletBlockWaiter()
			continue
		}

		if err := w.AttachBlock(block); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("walletUpdater AttachBlock stop")
			return
		}
	}
}

//RescanBlocks provide a trigger to rescan blocks
func (w *Wallet) RescanBlocks() {
	select {
	case w.rescanCh <- struct{}{}:
	default:
		return
	}
}

// DeleteAccount deletes account matching accountID, then rescan wallet
func (w *Wallet) DeleteAccount(accountID string) (err error) {
	w.rw.Lock()
	defer w.rw.Unlock()

	if err := w.AccountMgr.DeleteAccount(accountID); err != nil {
		return err
	}

	w.store.DeleteWalletTransactions()
	w.RescanBlocks()
	return nil
}

func (w *Wallet) UpdateAccountAlias(accountID string, newAlias string) (err error) {
	w.rw.Lock()
	defer w.rw.Unlock()

	if err := w.AccountMgr.UpdateAccountAlias(accountID, newAlias); err != nil {
		return err
	}

	w.store.DeleteWalletTransactions()
	w.RescanBlocks()
	return nil
}

func (w *Wallet) getRescanNotification() {
	select {
	case <-w.rescanCh:
		w.setRescanStatus()
	default:
		return
	}
}

func (w *Wallet) setRescanStatus() {
	block, _ := w.chain.GetBlockByHeight(0)
	w.Status.WorkHash = bc.Hash{}
	w.AttachBlock(block)
}

func (w *Wallet) walletBlockWaiter() {
	select {
	case <-w.chain.BlockWaiter(w.Status.WorkHeight + 1):
	case <-w.rescanCh:
		w.setRescanStatus()
	}
}

// GetWalletStatusInfo return current wallet StatusInfo
func (w *Wallet) GetWalletStatusInfo() StatusInfo {
	w.rw.RLock()
	defer w.rw.RUnlock()

	return w.Status
}
