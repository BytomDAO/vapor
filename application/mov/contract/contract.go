package contract

import (
	"encoding/hex"

	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/vm"
)

const (
	sizeOfCancelClauseArgs         = 3
	sizeOfPartialTradeClauseArgsV1 = 3
	sizeOfFullTradeClauseArgsV1    = 2
	sizeOfPartialTradeClauseArgsV2 = 4
	sizeOfFullTradeClauseArgsV2    = 3
)

// smart contract clause select for different unlock method
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
	if len(input.Arguments()) == sizeOfPartialTradeClauseArgsV1 || len(input.Arguments()) == sizeOfPartialTradeClauseArgsV2 {
		return hex.EncodeToString(input.Arguments()[len(input.Arguments())-1]) == hex.EncodeToString(vm.Int64Bytes(PartialTradeClauseSelector))
	}
	return false
}

// IsFullTradeClauseSelector check if input select full trade clause
func IsFullTradeClauseSelector(input *types.TxInput) bool {
	if len(input.Arguments()) == sizeOfFullTradeClauseArgsV1 || len(input.Arguments()) == sizeOfFullTradeClauseArgsV2 {
		return hex.EncodeToString(input.Arguments()[len(input.Arguments())-1]) == hex.EncodeToString(vm.Int64Bytes(FullTradeClauseSelector))
	}
	return false
}

// FeeRate return the rate of fee from input witness
func FeeRate(input *types.TxInput) (int64, error) {
	if IsFullTradeClauseSelector(input) {
		if len(input.Arguments()) == sizeOfFullTradeClauseArgsV1 {
			return 10, nil
		}
		return vm.AsInt64(input.Arguments()[0])
	}
	if IsPartialTradeClauseSelector(input) {
		if len(input.Arguments()) == sizeOfPartialTradeClauseArgsV1 {
			return 10, nil
		}
		return vm.AsInt64(input.Arguments()[1])
	}
	return 0, errors.New("invalid trade input")
}
