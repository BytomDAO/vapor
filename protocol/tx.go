package protocol

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
	"github.com/bytom/vapor/protocol/validation"
)

// GetTransactionStatus return the transaction status of give block
func (c *Chain) GetTransactionStatus(hash *bc.Hash) (*bc.TransactionStatus, error) {
	return c.store.GetTransactionStatus(hash)
}

// GetTransactionsUtxo return all the utxos that related to the txs' inputs
func (c *Chain) GetTransactionsUtxo(view *state.UtxoViewpoint, txs []*bc.Tx) error {
	return c.store.GetTransactionsUtxo(view, txs)
}

// ValidateTx validates the given transaction. A cache holds
// per-transaction validation results and is consulted before
// performing full validation.
func (c *Chain) ValidateTx(tx *types.Tx) (bool, error) {
	if c.hasSeenTx(tx) {
		return false, nil
	}

	bh := c.BestBlockHeader()
	isOrphan, err := c.validateTx(tx, bh)
	if err == nil {
		c.markTransactions(tx)
	}
	return isOrphan, err
}

// validateTx validates the given transaction without checking duplication.
func (c *Chain) validateTx(tx *types.Tx, bh *types.BlockHeader) (bool, error) {
	if ok := c.txPool.HaveTransaction(&tx.ID); ok {
		return false, c.txPool.GetErrCache(&tx.ID)
	}

	if c.txPool.IsDust(tx) {
		c.txPool.AddErrCache(&tx.ID, ErrDustTx)
		return false, ErrDustTx
	}

	gasStatus, err := validation.ValidateTx(tx.Tx, types.MapBlock(&types.Block{BlockHeader: *bh}))
	if !gasStatus.GasValid {
		c.txPool.AddErrCache(&tx.ID, err)
		return false, err
	}

	txVerifyResult := &bc.TxVerifyResult{StatusFail: err != nil}
	for _, p := range c.subProtocols {
		if err := p.ValidateTx(tx, txVerifyResult, bh.Height); err != nil {
			c.txPool.AddErrCache(&tx.ID, err)
			return false, err
		}
	}

	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "tx_id": tx.Tx.ID.String(), "error": err}).Info("transaction status fail")
	}

	return c.txPool.ProcessTransaction(tx, err != nil, bh.Height, gasStatus.BTMValue)
}
