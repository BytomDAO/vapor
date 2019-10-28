package contract

import (
	"encoding/hex"

	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm"
)

const (
	sizeOfCancelClauseArgs = 3
	sizeOfPartialTradeClauseArgs = 3
	sizeOfFullTradeClauseArgs = 2
)

const (
	PartialTradeClauseSelector int64 = iota
	FullTradeClauseSelector
	CancelClauseSelector
)

func IsCancelClauseSelector(input *types.TxInput) bool {
	return len(input.Arguments()) == sizeOfCancelClauseArgs && hex.EncodeToString(input.Arguments()[len(input.Arguments()) - 1]) == hex.EncodeToString(vm.Int64Bytes(CancelClauseSelector))
}

func IsTradeClauseSelector(input *types.TxInput) bool {
	return IsPartialTradeClauseSelector(input) || IsFullTradeClauseSelector(input)
}

func IsPartialTradeClauseSelector(input *types.TxInput) bool {
	return len(input.Arguments()) == sizeOfPartialTradeClauseArgs && hex.EncodeToString(input.Arguments()[len(input.Arguments()) - 1]) == hex.EncodeToString(vm.Int64Bytes(PartialTradeClauseSelector))
}

func IsFullTradeClauseSelector(input *types.TxInput) bool {
	return len(input.Arguments()) == sizeOfFullTradeClauseArgs && hex.EncodeToString(input.Arguments()[len(input.Arguments()) - 1]) == hex.EncodeToString(vm.Int64Bytes(FullTradeClauseSelector))
}
