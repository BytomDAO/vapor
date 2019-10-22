package util

import (
	"encoding/hex"

	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm"
)

func IsCancelClauseSelector(input *types.TxInput) bool {
	return len(input.Arguments()) == 3 && hex.EncodeToString(input.Arguments()[2]) == hex.EncodeToString(vm.Int64Bytes(2))
}

func IsTradeClauseSelector(input *types.TxInput) bool {
	return IsPartialTradeClauseSelector(input) || IsFullTradeClauseSelector(input)
}

func IsPartialTradeClauseSelector(input *types.TxInput) bool {
	return len(input.Arguments()) == 3 && hex.EncodeToString(input.Arguments()[2]) == hex.EncodeToString(vm.Int64Bytes(0))
}

func IsFullTradeClauseSelector(input *types.TxInput) bool {
	return len(input.Arguments()) == 2 && hex.EncodeToString(input.Arguments()[1]) == hex.EncodeToString(vm.Int64Bytes(1))
}

func GetTradeReceivePosition(input *types.TxInput) (int64, error) {
	if IsPartialTradeClauseSelector(input) {
		return vm.AsInt64(input.Arguments()[1])
	}

	if IsFullTradeClauseSelector(input) {
		return vm.AsInt64(input.Arguments()[0])
	}
	return 0, errors.New("non trade transaction input")
}
