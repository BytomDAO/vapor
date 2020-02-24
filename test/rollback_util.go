package test

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/vapor/account"
	"github.com/bytom/vapor/blockchain/txbuilder"
	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/database"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/database/storage"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
	"github.com/bytom/vapor/protocol/validation"
	"github.com/bytom/vapor/protocol/vm/vmutil"
)

const (
	logModule     = "mining"
	batchApplyNum = 64

	timeoutOk = iota + 1
	timeoutWarn
	timeoutCritical
)

type byTime []*protocol.TxDesc

func (a byTime) Len() int           { return len(a) }
func (a byTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byTime) Less(i, j int) bool { return a[i].Added.Before(a[j].Added) }

// NewBlockTemplate returns a new block template that is ready to be solved
func NewBlockTemplate(chain *protocol.Chain, db dbm.DB, accountManager *account.Manager, timestamp uint64, warnDuration, criticalDuration time.Duration) (*types.Block, error) {
	fmt.Println("NewBlockTemplate")
	builder := newBlockBuilder(chain, db, accountManager, timestamp, warnDuration, criticalDuration)
	return builder.build()
}

type blockBuilder struct {
	chain          *protocol.Chain
	db             dbm.DB
	accountManager *account.Manager

	block    *types.Block
	txStatus *bc.TransactionStatus
	utxoView *state.UtxoViewpoint

	warnTimeoutCh     <-chan time.Time
	criticalTimeoutCh <-chan time.Time
	timeoutStatus     uint8
	gasLeft           int64
}

func newBlockBuilder(chain *protocol.Chain, db dbm.DB, accountManager *account.Manager, timestamp uint64, warnDuration, criticalDuration time.Duration) *blockBuilder {
	preBlockHeader := chain.BestBlockHeader()
	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:           1,
			Height:            preBlockHeader.Height + 1,
			PreviousBlockHash: preBlockHeader.Hash(),
			Timestamp:         timestamp,
			BlockCommitment:   types.BlockCommitment{},
			BlockWitness:      types.BlockWitness{Witness: make([][]byte, consensus.ActiveNetParams.NumOfConsensusNode)},
		},
	}

	builder := &blockBuilder{
		chain:             chain,
		db:                db,
		accountManager:    accountManager,
		block:             block,
		txStatus:          bc.NewTransactionStatus(),
		utxoView:          state.NewUtxoViewpoint(),
		warnTimeoutCh:     time.After(warnDuration),
		criticalTimeoutCh: time.After(criticalDuration),
		gasLeft:           int64(consensus.ActiveNetParams.MaxBlockGas),
		timeoutStatus:     timeoutOk,
	}
	return builder
}

func (b *blockBuilder) applyCoinbaseTransaction() error {
	coinbaseTx, err := b.createCoinbaseTx()
	if err != nil {
		return errors.Wrap(err, "fail on create coinbase tx")
	}

	gasState, err := validation.ValidateTx(coinbaseTx.Tx, &bc.Block{BlockHeader: &bc.BlockHeader{Height: b.block.Height}, Transactions: []*bc.Tx{coinbaseTx.Tx}})
	if err != nil {
		return err
	}

	b.block.Transactions = append(b.block.Transactions, coinbaseTx)
	if err := b.txStatus.SetStatus(0, false); err != nil {
		return err
	}

	b.gasLeft -= gasState.GasUsed
	return nil
}

func (b *blockBuilder) applyVoteTransaction() error {
	tx, err := b.createVoteTx(b.accountManager, b.block.Height)

	if err != nil {
		return errors.Wrap(err, "fail on create vote tx")
	}

	gasState, err := validation.ValidateTx(tx.Tx, &bc.Block{BlockHeader: &bc.BlockHeader{Height: b.block.Height}, Transactions: []*bc.Tx{tx.Tx}})
	if err != nil {
		return err
	}

	b.block.Transactions = append(b.block.Transactions, tx)
	if err := b.txStatus.SetStatus(1, false); err != nil {
		return err
	}
	b.gasLeft -= gasState.GasUsed

	batch := b.db.NewBatch()
	view := &state.UtxoViewpoint{
		Entries: map[bc.Hash]*storage.UtxoEntry{
			tx.Tx.SpentOutputIDs[0]: &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: false},
		},
	}
	//fmt.Println("[important] applyVoteTransaction go to save SpentOutputIDs", tx.Tx.SpentOutputIDs[0].String())
	if err := database.SaveUtxoView(batch, view); err != nil {
		return err
	}
	batch.Write()
	//fmt.Println("[important] batch write")

	// prevout := tx.Tx.SpentOutputIDs[0]
	// utxoEntry, err := database.GetUtxo(b.db, &prevout)
	// fmt.Println("[important] utxoEntry:", utxoEntry.String(), "err", err)
	// txs := []*types.Tx{}
	// txs = append(txs, tx)
	// if err := b.applyTransactions(txs, timeoutWarn); err != nil {
	// 	return err
	// }

	return err
}

func (b *blockBuilder) applyTransactions(txs []*types.Tx, timeoutStatus uint8) error {
	tempTxs := []*types.Tx{}
	for i := 0; i < len(txs); i++ {
		if tempTxs = append(tempTxs, txs[i]); len(tempTxs) < batchApplyNum && i != len(txs)-1 {
			continue
		}

		fmt.Println("test", "i=", i, "preValidateTxs")
		results, gasLeft := preValidateTxs(tempTxs, b.chain, b.utxoView, b.gasLeft)
		for _, result := range results {
			if result.err != nil && !result.gasOnly {
				log.WithFields(log.Fields{"module": logModule, "error": result.err}).Error("mining block generation: skip tx due to")
				b.chain.GetTxPool().RemoveTransaction(&result.tx.ID)
				continue
			}

			if err := b.txStatus.SetStatus(len(b.block.Transactions), result.gasOnly); err != nil {
				return err
			}

			b.block.Transactions = append(b.block.Transactions, result.tx)
		}

		b.gasLeft = gasLeft
		tempTxs = []*types.Tx{}
		if b.getTimeoutStatus() >= timeoutStatus {
			break
		}
	}
	return nil
}

func (b *blockBuilder) applyTransactionFromPool() error {
	txDescList := b.chain.GetTxPool().GetTransactions()
	sort.Sort(byTime(txDescList))

	poolTxs := make([]*types.Tx, len(txDescList))
	for i, txDesc := range txDescList {
		poolTxs[i] = txDesc.Tx
	}

	return b.applyTransactions(poolTxs, timeoutWarn)
}

func (b *blockBuilder) applyTransactionFromSubProtocol() error {
	cp, err := b.accountManager.GetCoinbaseControlProgram()
	if err != nil {
		return err
	}

	isTimeout := func() bool {
		return b.getTimeoutStatus() > timeoutOk
	}

	for i, p := range b.chain.SubProtocols() {
		if b.gasLeft <= 0 || isTimeout() {
			break
		}

		subTxs, err := p.BeforeProposalBlock(b.block.Transactions, cp, b.block.Height, b.gasLeft, isTimeout)
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "index": i, "error": err}).Error("failed on sub protocol txs package")
			continue
		}

		if err := b.applyTransactions(subTxs, timeoutCritical); err != nil {
			return err
		}
	}
	return nil
}

func (b *blockBuilder) build() (*types.Block, error) {
	if err := b.applyCoinbaseTransaction(); err != nil {
		return nil, err
	}

	if err := b.applyVoteTransaction(); err != nil {
		return nil, err
	}

	if err := b.applyTransactionFromPool(); err != nil {
		return nil, err
	}

	if err := b.applyTransactionFromSubProtocol(); err != nil {
		return nil, err
	}

	if err := b.calcBlockCommitment(); err != nil {
		return nil, err
	}

	if err := b.chain.SignBlockHeader(&b.block.BlockHeader); err != nil {
		return nil, err
	}

	return b.block, nil
}

func (b *blockBuilder) calcBlockCommitment() (err error) {
	var txEntries []*bc.Tx
	for _, tx := range b.block.Transactions {
		txEntries = append(txEntries, tx.Tx)
	}

	b.block.BlockHeader.BlockCommitment.TransactionsMerkleRoot, err = types.TxMerkleRoot(txEntries)
	if err != nil {
		return err
	}

	b.block.BlockHeader.BlockCommitment.TransactionStatusHash, err = types.TxStatusMerkleRoot(b.txStatus.VerifyStatus)
	return err
}

// createCoinbaseTx returns a coinbase transaction paying an appropriate subsidy
// based on the passed block height to the provided address.  When the address
// is nil, the coinbase transaction will instead be redeemable by anyone.
func (b *blockBuilder) createCoinbaseTx() (*types.Tx, error) {
	consensusResult, err := b.chain.GetConsensusResultByHash(&b.block.PreviousBlockHash)
	if err != nil {
		return nil, err
	}

	rewards, err := consensusResult.GetCoinbaseRewards(b.block.Height - 1)
	if err != nil {
		return nil, err
	}

	return createCoinbaseTxByReward(b.accountManager, b.block.Height, rewards)
}

func (b *blockBuilder) createVoteTx(accountManager *account.Manager, blockHeight uint64) (*types.Tx, error) {
	testXpub, _ := hex.DecodeString("f3f6bcf61b65fa9d1566455a5688ca8b395efdc22e654963134b5e5cb0a45d8be522d21abc384a73177a7b9d64eba915fcfe2862d86a508a3c46dc410bdd72ad")

	arbitrary := append([]byte{0x00}, []byte(strconv.FormatUint(blockHeight, 10))...)
	var script []byte
	var err error
	if accountManager == nil {
		script, err = vmutil.DefaultCoinbaseProgram()
	} else {
		script, err = accountManager.GetCoinbaseControlProgram()
		arbitrary = append(arbitrary, accountManager.GetCoinbaseArbitrary()...)
	}
	if err != nil {
		return nil, err
	}

	if len(arbitrary) > consensus.ActiveNetParams.CoinbaseArbitrarySizeLimit {
		return nil, validation.ErrCoinbaseArbitraryOversize
	}

	builder := txbuilder.NewBuilder(time.Now())
	// if err = builder.AddInput(types.NewVetoInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, 100000000, 0, []byte{0x51}, testXpub), &txbuilder.SigningInstruction{}); err != nil {
	// 	return nil, err
	// }
	// if err = builder.AddOutput(types.NewIntraChainOutput(*consensus.BTMAssetID, 100000000, script)); err != nil {
	// 	return nil, err
	// }

	if err := builder.AddInput(types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 100000000, 0, script), &txbuilder.SigningInstruction{}); err != nil {
		return nil, err
	}
	if err := builder.AddOutput(types.NewVoteOutput(*consensus.BTMAssetID, 100000000, script, testXpub)); err != nil {
		return nil, err
	}

	_, txData, err := builder.Build()
	if err != nil {
		return nil, err
	}

	byteData, err := txData.MarshalText()
	if err != nil {
		return nil, err
	}

	txData.SerializedSize = uint64(len(byteData))
	tx := &types.Tx{
		TxData: *txData,
		Tx:     types.MapTx(txData),
	}

	fmt.Println("tx.Tx.String", tx.Tx.String())
	//a := tx.Tx.SpentOutputIDs
	for _, prevout := range tx.Tx.SpentOutputIDs {
		fmt.Println("createVoteTx SpentOutputIDs", prevout.String())
	}

	return tx, nil
}

func (b *blockBuilder) getTimeoutStatus() uint8 {
	if b.timeoutStatus == timeoutCritical {
		return b.timeoutStatus
	}

	select {
	case <-b.criticalTimeoutCh:
		b.timeoutStatus = timeoutCritical
	case <-b.warnTimeoutCh:
		b.timeoutStatus = timeoutWarn
	default:
	}

	return b.timeoutStatus
}

func createCoinbaseTxByReward(accountManager *account.Manager, blockHeight uint64, rewards []state.CoinbaseReward) (tx *types.Tx, err error) {
	arbitrary := append([]byte{0x00}, []byte(strconv.FormatUint(blockHeight, 10))...)
	var script []byte
	if accountManager == nil {
		script, err = vmutil.DefaultCoinbaseProgram()
	} else {
		script, err = accountManager.GetCoinbaseControlProgram()
		arbitrary = append(arbitrary, accountManager.GetCoinbaseArbitrary()...)
	}

	if err != nil {
		return nil, err
	}

	if len(arbitrary) > consensus.ActiveNetParams.CoinbaseArbitrarySizeLimit {
		return nil, validation.ErrCoinbaseArbitraryOversize
	}

	builder := txbuilder.NewBuilder(time.Now())
	if err = builder.AddInput(types.NewCoinbaseInput(arbitrary), &txbuilder.SigningInstruction{}); err != nil {
		return nil, err
	}

	if err = builder.AddOutput(types.NewIntraChainOutput(*consensus.BTMAssetID, 0, script)); err != nil {
		return nil, err
	}

	for _, r := range rewards {
		if err = builder.AddOutput(types.NewIntraChainOutput(*consensus.BTMAssetID, r.Amount, r.ControlProgram)); err != nil {
			return nil, err
		}
	}

	_, txData, err := builder.Build()
	if err != nil {
		return nil, err
	}

	byteData, err := txData.MarshalText()
	if err != nil {
		return nil, err
	}

	txData.SerializedSize = uint64(len(byteData))
	tx = &types.Tx{
		TxData: *txData,
		Tx:     types.MapTx(txData),
	}
	return tx, nil
}

type validateTxResult struct {
	tx      *types.Tx
	gasOnly bool
	err     error
}

func preValidateTxs(txs []*types.Tx, chain *protocol.Chain, view *state.UtxoViewpoint, gasLeft int64) ([]*validateTxResult, int64) {
	var results []*validateTxResult
	bcBlock := &bc.Block{BlockHeader: &bc.BlockHeader{Height: chain.BestBlockHeight() + 1}}
	bcTxs := make([]*bc.Tx, len(txs))
	for i, tx := range txs {
		bcTxs[i] = tx.Tx
	}

	validateResults := validation.ValidateTxs(bcTxs, bcBlock)
	for i := 0; i < len(validateResults) && gasLeft > 0; i++ {
		gasOnlyTx := false
		gasStatus := validateResults[i].GetGasState()
		if err := validateResults[i].GetError(); err != nil {
			if !gasStatus.GasValid {
				results = append(results, &validateTxResult{tx: txs[i], err: err})
				continue
			}
			gasOnlyTx = true
		}

		if err := chain.GetTransactionsUtxo(view, []*bc.Tx{bcTxs[i]}); err != nil {
			results = append(results, &validateTxResult{tx: txs[i], err: err})
			continue
		}

		if gasLeft-gasStatus.GasUsed < 0 {
			break
		}

		if err := view.ApplyTransaction(bcBlock, bcTxs[i], gasOnlyTx); err != nil {
			results = append(results, &validateTxResult{tx: txs[i], err: err})
			continue
		}

		if err := validateBySubProtocols(txs[i], validateResults[i].GetError() != nil, chain.SubProtocols()); err != nil {
			results = append(results, &validateTxResult{tx: txs[i], err: err})
			continue
		}

		results = append(results, &validateTxResult{tx: txs[i], gasOnly: gasOnlyTx, err: validateResults[i].GetError()})
		gasLeft -= gasStatus.GasUsed
	}
	return results, gasLeft
}

func validateBySubProtocols(tx *types.Tx, statusFail bool, subProtocols []protocol.Protocoler) error {
	for _, subProtocol := range subProtocols {
		verifyResult := &bc.TxVerifyResult{StatusFail: statusFail}
		if err := subProtocol.ValidateTx(tx, verifyResult); err != nil {
			return err
		}
	}
	return nil
}
