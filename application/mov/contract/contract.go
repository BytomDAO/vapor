package contract

import (
	"encoding/hex"

	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/vm"
)

const (
	sizeOfCancelClauseArgs       = 3
	sizeOfPartialTradeClauseArgs = 3
	sizeOfFullTradeClauseArgs    = 2
)

// smart contract clause select for differnet unlock method
const (
	PartialTradeClauseSelector int64 = iota
	FullTradeClauseSelector
	CancelClauseSelector
)

// IsCancelClauseSelector check if input select cancel clause
func IsCancelClauseSelector(input *types.TxInput) bool {
	return len(input.Arguments()) == sizeOfCancelClauseArgs && hex.EncodeToString(input.Arguments()[len(input.Arguments())-1]) == hex.EncodeToString(vm.Int64Bytes(CancelClauseSelector))
}

// IsTradeClauseSelector check if input select is partial trade clause or full trade clause
func IsTradeClauseSelector(input *types.TxInput) bool {
	return IsPartialTradeClauseSelector(input) || IsFullTradeClauseSelector(input)
}

// IsPartialTradeClauseSelector check if input select partial trade clause
func IsPartialTradeClauseSelector(input *types.TxInput) bool {
	return len(input.Arguments()) == sizeOfPartialTradeClauseArgs && hex.EncodeToString(input.Arguments()[len(input.Arguments())-1]) == hex.EncodeToString(vm.Int64Bytes(PartialTradeClauseSelector))
}

// IsFullTradeClauseSelector check if input select full trade clause
func IsFullTradeClauseSelector(input *types.TxInput) bool {
	return len(input.Arguments()) == sizeOfFullTradeClauseArgs && hex.EncodeToString(input.Arguments()[len(input.Arguments())-1]) == hex.EncodeToString(vm.Int64Bytes(FullTradeClauseSelector))
}
