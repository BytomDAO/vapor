package util

import (
	"encoding/hex"

	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm"
)

const (
	sizeOfCancelClauseArgs = 3
	sizeOfPartialTradeClauseArgs = 3
	sizeOfFullTradeClauseArgs = 2

	partialTradeClauseSelector int64 = iota
	fullTradeClauseSelector
	cancelClauseSelector
)

func IsCancelClauseSelector(input *types.TxInput) bool {
	return len(input.Arguments()) == sizeOfCancelClauseArgs && hex.EncodeToString(input.Arguments()[len(input.Arguments()) - 1]) == hex.EncodeToString(vm.Int64Bytes(cancelClauseSelector))
}

func IsTradeClauseSelector(input *types.TxInput) bool {
	return IsPartialTradeClauseSelector(input) || IsFullTradeClauseSelector(input)
}

func IsPartialTradeClauseSelector(input *types.TxInput) bool {
	return len(input.Arguments()) == sizeOfPartialTradeClauseArgs && hex.EncodeToString(input.Arguments()[len(input.Arguments()) - 1]) == hex.EncodeToString(vm.Int64Bytes(partialTradeClauseSelector))
}

func IsFullTradeClauseSelector(input *types.TxInput) bool {
	return len(input.Arguments()) == sizeOfFullTradeClauseArgs && hex.EncodeToString(input.Arguments()[len(input.Arguments()) - 1]) == hex.EncodeToString(vm.Int64Bytes(fullTradeClauseSelector))
}
