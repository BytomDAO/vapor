package wallet

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/vapor/account"
	"github.com/bytom/vapor/asset"
	"github.com/bytom/vapor/blockchain/pseudohsm"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/event"
	"github.com/bytom/vapor/protocol"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
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
	ErrGetAsset                = errors.New("Failed to find asset definition")
	ErrAccntTxIDNotFound       = errors.New("account TXID not found")
	ErrGetStandardUTXO         = errors.New("failed get standard UTXO")
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
	Store           WalletStore
	rw              sync.RWMutex
	Status          StatusInfo
	TxIndexFlag     bool
	AccountMgr      *account.Manager
	AssetReg        *asset.Registry
	Hsm             *pseudohsm.HSM
	Chain           *protocol.Chain
	RecoveryMgr     *recoveryManager
	EventDispatcher *event.Dispatcher
	TxMsgSub        *event.Subscription

	rescanCh chan struct{}
}

//NewWallet return a new wallet instance
func NewWallet(store WalletStore, account *account.Manager, asset *asset.Registry, hsm *pseudohsm.HSM, chain *protocol.Chain, dispatcher *event.Dispatcher, txIndexFlag bool) (*Wallet, error) {
	w := &Wallet{
		Store:           store,
		AccountMgr:      account,
		AssetReg:        asset,
		Chain:           chain,
		Hsm:             hsm,
		RecoveryMgr:     NewRecoveryManager(store, account),
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

	return w, nil
}

// Run go to run some wallet recorvery and clean tx thread
func (w *Wallet) Run() error {
	var err error
	w.TxMsgSub, err = w.EventDispatcher.Subscribe(protocol.TxMsgEvent{})
	if err != nil {
		return err
	}

	go w.walletUpdater()
	go w.delUnconfirmedTx()
	go w.MemPoolTxQueryLoop()

	return nil
}

// MemPoolTxQueryLoop constantly pass a transaction accepted by mempool to the wallet.
func (w *Wallet) MemPoolTxQueryLoop() {
	for {
		select {
		case obj, ok := <-w.TxMsgSub.Chan():
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
	} else if !w.Chain.BlockExist(&w.Status.BestHash) {
		return errBestBlockNotFoundInCore
	}

	return nil
}

//LoadWalletInfo return stored wallet info and nil,
//if error, return initial wallet info and err
func (w *Wallet) LoadWalletInfo() error {
	walletStatus, err := w.Store.GetWalletInfo()
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
		w.Store.DeleteWalletTransactions()
		w.Store.DeleteWalletUTXOs()
	}

	w.Status.Version = currentVersion
	w.Status.WorkHash = bc.Hash{}
	block, err := w.Chain.GetBlockByHeight(0)
	if err != nil {
		return err
	}

	return w.AttachBlock(block)
}

func (w *Wallet) commitWalletInfo(store WalletStore) error {
	if err := store.SetWalletInfo(&w.Status); err != nil {
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
	txStatus, err := w.Chain.GetTransactionStatus(&blockHash)
	if err != nil {
		return err
	}

	if err := w.RecoveryMgr.FilterRecoveryTxs(block); err != nil {
		log.WithField("err", err).Error("filter recovery txs")
		w.RecoveryMgr.finished()
	}

	annotatedTxs := w.filterAccountTxs(block, txStatus)
	if err := saveExternalAssetDefinition(block, w.Store); err != nil {
		return err
	}

	w.annotateTxsAccount(annotatedTxs)

	newStore := w.Store.InitBatch()
	if err := w.indexTransactions(block, txStatus, annotatedTxs, newStore); err != nil {
		return err
	}

	w.attachUtxos(block, txStatus, newStore)
	w.Status.WorkHeight = block.Height
	w.Status.WorkHash = block.Hash()
	if w.Status.WorkHeight >= w.Status.BestHeight {
		w.Status.BestHeight = w.Status.WorkHeight
		w.Status.BestHash = w.Status.WorkHash
	}

	if err := w.commitWalletInfo(newStore); err != nil {
		return err
	}

	if err := newStore.CommitBatch(); err != nil {
		return err
	}

	return nil
}

// DetachBlock detach a block and rollback state
func (w *Wallet) DetachBlock(block *types.Block) error {
	w.rw.Lock()
	defer w.rw.Unlock()

	blockHash := block.Hash()
	txStatus, err := w.Chain.GetTransactionStatus(&blockHash)
	if err != nil {
		return err
	}

	newStore := w.Store.InitBatch()

	w.detachUtxos(block, txStatus, newStore)
	newStore.DeleteTransactions(w.Status.BestHeight)

	w.Status.BestHeight = block.Height - 1
	w.Status.BestHash = block.PreviousBlockHash

	if w.Status.WorkHeight > w.Status.BestHeight {
		w.Status.WorkHeight = w.Status.BestHeight
		w.Status.WorkHash = w.Status.BestHash
	}
	if err := w.commitWalletInfo(newStore); err != nil {
		return err
	}

	if err := newStore.CommitBatch(); err != nil {
		return err
	}

	return nil
}

//WalletUpdate process every valid block and reverse every invalid block which need to rollback
func (w *Wallet) walletUpdater() {
	for {
		w.getRescanNotification()
		for !w.Chain.InMainChain(w.Status.BestHash) {
			block, err := w.Chain.GetBlockByHash(&w.Status.BestHash)
			if err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("walletUpdater GetBlockByHash")
				return
			}

			if err := w.DetachBlock(block); err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("walletUpdater detachBlock stop")
				return
			}
		}

		block, _ := w.Chain.GetBlockByHeight(w.Status.WorkHeight + 1)
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

	w.Store.DeleteWalletTransactions()
	w.RescanBlocks()
	return nil
}

// Rollback wallet to target height
func (w *Wallet) Rollback(targetHeight uint64) error {
	for w.Status.WorkHeight > targetHeight {
		block, err := w.Chain.GetBlockByHash(&w.Status.WorkHash)
		if err != nil {
			return err
		}

		if err = w.DetachBlock(block); err != nil {
			return err
		}
	}

	return nil
}

func (w *Wallet) UpdateAccountAlias(accountID string, newAlias string) (err error) {
	w.rw.Lock()
	defer w.rw.Unlock()

	if err := w.AccountMgr.UpdateAccountAlias(accountID, newAlias); err != nil {
		return err
	}

	w.Store.DeleteWalletTransactions()
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
	block, _ := w.Chain.GetBlockByHeight(0)
	w.Status.WorkHash = bc.Hash{}
	w.AttachBlock(block)
}

func (w *Wallet) walletBlockWaiter() {
	select {
	case <-w.Chain.BlockWaiter(w.Status.WorkHeight + 1):
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
