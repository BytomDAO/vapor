package common

import (
	"github.com/vapor/application/mov/contract"
	"github.com/vapor/consensus/segwit"
	"github.com/vapor/protocol/bc/types"
)

func IsMatchedTx(tx *types.Tx) bool {
	if len(tx.Inputs) < 2 {
		return false
	}
	for _, input := range tx.Inputs {
		if input.InputType() == types.SpendInputType && contract.IsTradeClauseSelector(input) && segwit.IsP2WMCScript(input.ControlProgram()) {
			return true
		}
	}
	return false
}

func IsCancelOrderTx(tx *types.Tx) bool {
	for _, input := range tx.Inputs {
		if input.InputType() == types.SpendInputType && contract.IsCancelClauseSelector(input) && segwit.IsP2WMCScript(input.ControlProgram()) {
			return true
		}
	}
	return false
}

