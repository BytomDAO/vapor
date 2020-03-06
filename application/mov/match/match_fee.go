package match

import (
	"math"

	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
)

var (
	// ErrAmountOfFeeExceedMaximum represent The fee charged is exceeded the maximum
	ErrAmountOfFeeExceedMaximum = errors.New("amount of fee greater than max fee amount")
)

// AllocatedAssets represent reallocated assets after calculating fees
type AllocatedAssets struct {
	Received []*bc.AssetAmount
	Refunds  []RefundAssets
	Fees     []*bc.AssetAmount
}

// RefundAssets represent alias for assetAmount array, because each transaction participant can be refunded multiple assets
type RefundAssets []*bc.AssetAmount

// FeeStrategy used to indicate how to charge a matching fee
type FeeStrategy interface {
	// Allocate will allocate the price differential in matching transaction to the participants and the fee
	// @param receiveAmounts the amount of assets that the participants in the matching transaction can received when no fee is considered
	// @param priceDiffs price differential of matching transaction
	// @return reallocated assets after calculating fees
	Allocate(receiveAmounts []*bc.AssetAmount, priceDiffs map[bc.AssetID]int64) *AllocatedAssets

	// Validate verify that the fee charged for a matching transaction is correct
	Validate(receiveAmounts []*bc.AssetAmount, priceDiffs, feeAmounts map[bc.AssetID]int64) error
}

// DefaultFeeStrategy represent the default fee charge strategy
type DefaultFeeStrategy struct {
	maxFeeRate float64
}

// NewDefaultFeeStrategy return a new instance of DefaultFeeStrategy
func NewDefaultFeeStrategy(maxFeeRate float64) *DefaultFeeStrategy {
	return &DefaultFeeStrategy{maxFeeRate: maxFeeRate}
}

// Allocate will allocate the price differential in matching transaction to the participants and the fee
func (d *DefaultFeeStrategy) Allocate(receiveAmounts []*bc.AssetAmount, priceDiffs map[bc.AssetID]int64) *AllocatedAssets {
	var feeAmounts []*bc.AssetAmount
	refundAmounts := make([]RefundAssets, len(receiveAmounts))

	for _, receiveAmount := range receiveAmounts {
		if _, ok := priceDiffs[*receiveAmount.AssetId]; !ok {
			continue
		}

		priceDiff := priceDiffs[*receiveAmount.AssetId]
		maxFeeAmount := calcMaxFeeAmount(receiveAmount.Amount, d.maxFeeRate)

		feeAmount, reminder := priceDiff, int64(0)
		if priceDiff > maxFeeAmount {
			feeAmount = maxFeeAmount
			reminder = priceDiff - maxFeeAmount
		}

		feeAmounts = append(feeAmounts, &bc.AssetAmount{AssetId: receiveAmount.AssetId, Amount: uint64(feeAmount)})

		// There is the remaining amount after paying the handling fee, assign it evenly to participants in the transaction
		averageAmount := reminder / int64(len(receiveAmounts))
		if averageAmount == 0 {
			averageAmount = 1
		}

		for i := 0; i < len(receiveAmounts) && reminder > 0; i++ {
			amount := averageAmount
			if i == len(receiveAmounts)-1 {
				amount = reminder
			}
			refundAmounts[i] = append(refundAmounts[i], &bc.AssetAmount{AssetId: receiveAmount.AssetId, Amount: uint64(amount)})
			reminder -= averageAmount
		}
	}

	receivedAfterDeductFee := make([]*bc.AssetAmount, len(receiveAmounts))
	copy(receivedAfterDeductFee, receiveAmounts)
	return &AllocatedAssets{Received: receivedAfterDeductFee, Refunds: refundAmounts, Fees: feeAmounts}
}

// Validate verify that the fee charged for a matching transaction is correct
func (d *DefaultFeeStrategy) Validate(receiveAmounts []*bc.AssetAmount, feeAmounts, priceDiffs map[bc.AssetID]int64) error {
	for _, receiveAmount := range receiveAmounts {
		if feeAmount, ok := feeAmounts[*receiveAmount.AssetId]; ok {
			if feeAmount > calcMaxFeeAmount(receiveAmount.Amount+uint64(priceDiffs[*receiveAmount.AssetId]), d.maxFeeRate) {
				return ErrAmountOfFeeExceedMaximum
			}
		}
	}
	return nil
}

func calcMaxFeeAmount(amount uint64, maxFeeRate float64) int64 {
	return int64(math.Ceil(float64(amount) * maxFeeRate))
}
