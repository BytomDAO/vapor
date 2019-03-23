package mining

import (
	"encoding/binary"
	"encoding/json"
	"sort"
	"strconv"
	"time"

	"github.com/vapor/protocol/vm"

	"github.com/vapor/common"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/account"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/config"
	"github.com/vapor/consensus"
	engine "github.com/vapor/consensus/consensus"
	dpos "github.com/vapor/consensus/consensus/dpos"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
	"github.com/vapor/protocol/validation"
	"github.com/vapor/protocol/vm/vmutil"
)

// createCoinbaseTx returns a coinbase transaction paying an appropriate subsidy
// based on the passed block height to the provided address.  When the address
// is nil, the coinbase transaction will instead be redeemable by anyone.
func createCoinbaseTx(accountManager *account.Manager, amount uint64, blockHeight uint64, delegateInfo interface{}, timestamp uint64) (tx *types.Tx, err error) {
	//amount += consensus.BlockSubsidy(blockHeight)
	arbitrary := append([]byte{0x00}, []byte(strconv.FormatUint(blockHeight, 10))...)

	var script []byte
	address, _ := common.DecodeAddress(config.CommonConfig.Consensus.Coinbase, &consensus.ActiveNetParams)
	redeemContract := address.ScriptAddress()
	script, _ = vmutil.P2WPKHProgram(redeemContract)

	if err != nil {
		return nil, err
	}

	if len(arbitrary) > consensus.CoinbaseArbitrarySizeLimit {
		return nil, validation.ErrCoinbaseArbitraryOversize
	}

	builder := txbuilder.NewBuilder(time.Now())
	if err = builder.AddInput(types.NewCoinbaseInput(arbitrary), &txbuilder.SigningInstruction{}); err != nil {
		return nil, err
	}

	if err = builder.AddOutput(types.NewTxOutput(*consensus.BTMAssetID, amount, script)); err != nil {
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
	delegates := dpos.DelegateInfoList{}
	if delegateInfo != nil {
		tmp := delegateInfo.(*dpos.DelegateInfo)
		delegates.Delegate = *tmp
	}

	var xPrv chainkd.XPrv
	if config.CommonConfig.Consensus.XPrv == "" {
		return nil, errors.New("Signer is empty")
	}
	xPrv.UnmarshalText([]byte(config.CommonConfig.Consensus.XPrv))

	buf := [8]byte{}
	binary.LittleEndian.PutUint64(buf[:], timestamp)
	delegates.SigTime = xPrv.Sign(buf[:])
	delegates.Xpub = xPrv.XPub()

	data, err := json.Marshal(&delegates)
	if err != nil {
		return nil, err
	}

	msg := dpos.DposMsg{
		Type: vm.OP_DELEGATE,
		Data: data,
	}

	data, err = json.Marshal(&msg)
	if err != nil {
		return nil, err
	}
	txData.ReferenceData = data

	tx = &types.Tx{
		TxData: *txData,
		Tx:     types.MapTx(txData),
	}
	return tx, nil
}

// NewBlockTemplate returns a new block template that is ready to be solved
func NewBlockTemplate(c *protocol.Chain, txPool *protocol.TxPool, accountManager *account.Manager, engine engine.Engine, delegateInfo interface{}, blockTime uint64) (b *types.Block, err error) {
	view := state.NewUtxoViewpoint()
	txStatus := bc.NewTransactionStatus()
	if err := txStatus.SetStatus(0, false); err != nil {
		return nil, err
	}
	txEntries := []*bc.Tx{nil}
	gasUsed := uint64(0)
	txFee := uint64(0)

	// get preblock info for generate next block
	preBlockHeader := c.BestBlockHeader()
	preBlockHash := preBlockHeader.Hash()
	nextBlockHeight := preBlockHeader.Height + 1

	header := types.BlockHeader{
		Version:           1,
		Height:            nextBlockHeight,
		PreviousBlockHash: preBlockHash,
		Timestamp:         blockTime,
		BlockCommitment:   types.BlockCommitment{},
	}

	b = &types.Block{}
	bcBlock := &bc.Block{BlockHeader: &bc.BlockHeader{Height: nextBlockHeight}}
	b.Transactions = []*types.Tx{nil}

	txs := txPool.GetTransactions()
	sort.Sort(byTime(txs))
	for _, txDesc := range txs {
		tx := txDesc.Tx.Tx
		gasOnlyTx := false

		if err := c.GetTransactionsUtxo(view, []*bc.Tx{tx}); err != nil {
			blkGenSkipTxForErr(txPool, &tx.ID, err)
			continue
		}
		gasStatus, err := validation.ValidateTx(tx, bcBlock)
		if err != nil {
			if !gasStatus.GasValid {
				blkGenSkipTxForErr(txPool, &tx.ID, err)
				continue
			}
			gasOnlyTx = true
		}

		if gasUsed+uint64(gasStatus.GasUsed) > consensus.MaxBlockGas {
			break
		}

		if err := view.ApplyTransaction(bcBlock, tx, gasOnlyTx); err != nil {
			blkGenSkipTxForErr(txPool, &tx.ID, err)
			continue
		}

		if err := txStatus.SetStatus(len(b.Transactions), gasOnlyTx); err != nil {
			return nil, err
		}

		b.Transactions = append(b.Transactions, txDesc.Tx)
		txEntries = append(txEntries, tx)
		gasUsed += uint64(gasStatus.GasUsed)
		txFee += txDesc.Fee
		if gasUsed == consensus.MaxBlockGas {
			break
		}
	}

	b.BlockHeader = header
	// creater coinbase transaction
	b.Transactions[0], err = createCoinbaseTx(accountManager, txFee, nextBlockHeight, delegateInfo, b.Timestamp)
	if err != nil {
		return nil, errors.Wrap(err, "fail on createCoinbaseTx")
	}
	txEntries[0] = b.Transactions[0].Tx

	b.BlockHeader.BlockCommitment.TransactionsMerkleRoot, err = types.TxMerkleRoot(txEntries)
	if err != nil {
		return nil, err
	}

	b.BlockHeader.BlockCommitment.TransactionStatusHash, err = types.TxStatusMerkleRoot(txStatus.VerifyStatus)
	return b, err
}

func blkGenSkipTxForErr(txPool *protocol.TxPool, txHash *bc.Hash, err error) {
	log.WithField("error", err).Error("mining block generation: skip tx due to")
	txPool.RemoveTransaction(txHash)
}
