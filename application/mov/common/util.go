package common

import (
	"github.com/bytom/vapor/application/mov/contract"
	"github.com/bytom/vapor/consensus/segwit"
	"github.com/bytom/vapor/protocol/bc/types"
)

// IsMatchedTx check if this transaction has trade mov order input
func IsMatchedTx(tx *types.Tx) bool {
	if len(tx.Inputs) < 2 {
		return false
	}
	for _, input := range tx.Inputs {
		if input.InputType() == types.SpendInputType && segwit.IsP2WMCScript(input.ControlProgram()) && contract.IsTradeClauseSelector(input) {
			return true
		}
	}
	return false
}

// IsCancelOrderTx check if this transaction has cancel mov order input
func IsCancelOrderTx(tx *types.Tx) bool {
	for _, input := range tx.Inputs {
		if input.InputType() == types.SpendInputType && segwit.IsP2WMCScript(input.ControlProgram()) && contract.IsCancelClauseSelector(input) {
			return true
		}
	}
	return false
}
