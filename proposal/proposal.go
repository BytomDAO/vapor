package proposal

import (
	"sort"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/account"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/consensus"
	"github.com/vapor/errors"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
	"github.com/vapor/protocol/validation"
	"github.com/vapor/protocol/vm/vmutil"
)

const logModule = "mining"

// createCoinbaseTx returns a coinbase transaction paying an appropriate subsidy
// based on the passed block height to the provided address.  When the address
// is nil, the coinbase transaction will instead be redeemable by anyone.
func createCoinbaseTx(accountManager *account.Manager, blockHeight uint64) (tx *types.Tx, err error) {
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

	if len(arbitrary) > consensus.CoinbaseArbitrarySizeLimit {
		return nil, validation.ErrCoinbaseArbitraryOversize
	}

	builder := txbuilder.NewBuilder(time.Now())
	if err = builder.AddInput(types.NewCoinbaseInput(arbitrary), &txbuilder.SigningInstruction{}); err != nil {
		return nil, err
	}
	if err = builder.AddOutput(types.NewIntraChainOutput(*consensus.BTMAssetID, 0, script)); err != nil {
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

	tx = &types.Tx{
		TxData: *txData,
		Tx:     types.MapTx(txData),
	}
	return tx, nil
}

// restructCoinbaseTx build coinbase transaction with aggregate outputs when it achieved the specified block height
func restructCoinbaseTx(tx *types.Tx, rewards []state.CoinbaseReward) error {
	if len(rewards) == 0 {
		return nil
	}

	for _, r := range rewards {
		tx.Outputs = append(tx.Outputs, types.NewIntraChainOutput(*consensus.BTMAssetID, r.Amount, r.ControlProgram))
	}

	byteData, err := tx.TxData.MarshalText()
	if err != nil {
		return err
	}

	tx.TxData.SerializedSize = uint64(len(byteData))
	tx.Tx = types.MapTx(&tx.TxData)
	return nil
}

// NewBlockTemplate returns a new block template that is ready to be solved
func NewBlockTemplate(c *protocol.Chain, txPool *protocol.TxPool, accountManager *account.Manager, timestamp uint64) (b *types.Block, err error) {
	view := state.NewUtxoViewpoint()
	txStatus := bc.NewTransactionStatus()
	if err := txStatus.SetStatus(0, false); err != nil {
		return nil, err
	}
	txEntries := []*bc.Tx{nil}
	gasUsed := uint64(0)

	// get preblock info for generate next block
	preBlockHeader := c.BestBlockHeader()
	preBlockHash := preBlockHeader.Hash()
	nextBlockHeight := preBlockHeader.Height + 1

	b = &types.Block{
		BlockHeader: types.BlockHeader{
			Version:           1,
			Height:            nextBlockHeight,
			PreviousBlockHash: preBlockHash,
			Timestamp:         timestamp,
			BlockCommitment:   types.BlockCommitment{},
			BlockWitness:      types.BlockWitness{Witness: make([][]byte, consensus.NumOfConsensusNode)},
		},
	}
	bcBlock := &bc.Block{BlockHeader: &bc.BlockHeader{Height: nextBlockHeight}}
	b.Transactions = []*types.Tx{nil}

	txs := txPool.GetTransactions()
	sort.Sort(byTime(txs))

	entriesTxs := []*bc.Tx{}
	for _, txDesc := range txs {
		entriesTxs = append(entriesTxs, txDesc.Tx.Tx)
	}

	validateResults := validation.ValidateTxs(entriesTxs, bcBlock)
	for i, validateResult := range validateResults {
		txDesc := txs[i]
		tx := txDesc.Tx.Tx
		gasOnlyTx := false

		gasStatus := validateResult.GetGasState()
		if validateResult.GetError() != nil {
			if !gasStatus.GasValid {
				blkGenSkipTxForErr(txPool, &tx.ID, err)
				continue
			}
			gasOnlyTx = true
		}

		if err := c.GetTransactionsUtxo(view, []*bc.Tx{tx}); err != nil {
			blkGenSkipTxForErr(txPool, &tx.ID, err)
			continue
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
		if gasUsed == consensus.MaxBlockGas {
			break
		}

	}

	// create coinbase transaction
	b.Transactions[0], err = createCoinbaseTx(accountManager, nextBlockHeight)
	if err != nil {
		return nil, errors.Wrap(err, "fail on createCoinbaseTx")
	}

	consensusResult, err := c.GetConsensusResultByHash(&preBlockHash)
	if err != nil {
		return nil, err
	}

	if err := consensusResult.AttachCoinbaseReward(b); err != nil {
		return nil, err
	}

	rewards, err := consensusResult.GetCoinbaseRewards(nextBlockHeight)
	if err != nil {
		return nil, err
	}

	// restruct coinbase transaction
	if err = restructCoinbaseTx(b.Transactions[0], rewards); err != nil {
		return nil, errors.Wrap(err, "fail on createCoinbaseTx")
	}

	txEntries[0] = b.Transactions[0].Tx
	b.BlockHeader.BlockCommitment.TransactionsMerkleRoot, err = types.TxMerkleRoot(txEntries)
	if err != nil {
		return nil, err
	}

	b.BlockHeader.BlockCommitment.TransactionStatusHash, err = types.TxStatusMerkleRoot(txStatus.VerifyStatus)

	_, err = c.SignBlock(b)
	return b, err
}

func blkGenSkipTxForErr(txPool *protocol.TxPool, txHash *bc.Hash, err error) {
	log.WithFields(log.Fields{"module": logModule, "error": err}).Error("mining block generation: skip tx due to")
	txPool.RemoveTransaction(txHash)
}
