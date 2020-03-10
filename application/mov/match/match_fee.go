package match

import (
	"math"

	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
)

var (
	// ErrAmountOfFeeOutOfRange represent The fee charged is out of range
	ErrAmountOfFeeOutOfRange = errors.New("amount of fee is out of range")
)

// AllocatedAssets represent reallocated assets after calculating fees
type AllocatedAssets struct {
	Receives []*bc.AssetAmount
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
	Allocate(receiveAmounts, priceDiffs []*bc.AssetAmount) *AllocatedAssets

	// Validate verify that the fee charged for a matching transaction is correct
	Validate(receiveAmounts []*bc.AssetAmount, feeAmounts map[bc.AssetID]uint64) error
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
func (d *DefaultFeeStrategy) Allocate(receiveAmounts, priceDiffs []*bc.AssetAmount) *AllocatedAssets {
	feeMap := make(map[bc.AssetID]uint64)
	for _, priceDiff := range priceDiffs {
		feeMap[*priceDiff.AssetId] = priceDiff.Amount
	}

	var fees []*bc.AssetAmount
	refunds := make([]RefundAssets, len(receiveAmounts))
	receives := make([]*bc.AssetAmount, len(receiveAmounts))

	for i, receiveAmount := range receiveAmounts {
		amount := receiveAmount.Amount
		minFeeAmount := calcMinFeeAmount(amount)
		receives[i] = &bc.AssetAmount{AssetId: receiveAmount.AssetId, Amount: amount - minFeeAmount}
		feeMap[*receiveAmount.AssetId] += minFeeAmount

		maxFeeAmount := calcMaxFeeAmount(amount, d.maxFeeRate)
		feeAmount, reminder := feeMap[*receiveAmount.AssetId], uint64(0)
		if feeAmount > maxFeeAmount {
			reminder = feeAmount - maxFeeAmount
			feeAmount = maxFeeAmount
		}

		if feeAmount > 0 {
			fees = append(fees, &bc.AssetAmount{AssetId: receiveAmount.AssetId, Amount: feeAmount})
		}

		// There is the remaining amount after paying the handling fee, assign it evenly to participants in the transaction
		averageAmount := reminder / uint64(len(receiveAmounts))
		if averageAmount == 0 {
			averageAmount = 1
		}

		for j := 0; j < len(receiveAmounts) && reminder > 0; j++ {
			refundAmount := averageAmount
			if j == len(receiveAmounts)-1 {
				refundAmount = reminder
			}
			refunds[j] = append(refunds[j], &bc.AssetAmount{AssetId: receiveAmount.AssetId, Amount: refundAmount})
			reminder -= averageAmount
		}
	}
	return &AllocatedAssets{Receives: receives, Refunds: refunds, Fees: fees}
}

// Validate verify that the fee charged for a matching transaction is correct
func (d *DefaultFeeStrategy) Validate(receiveAmounts []*bc.AssetAmount, feeAmounts map[bc.AssetID]uint64) error {
	for _, receiveAmount := range receiveAmounts {
		feeAmount := feeAmounts[*receiveAmount.AssetId]
		maxFeeAmount := calcMaxFeeAmount(receiveAmount.Amount, d.maxFeeRate)
		minFeeAmount := calcMinFeeAmount(receiveAmount.Amount)
		if feeAmount < minFeeAmount || feeAmount > maxFeeAmount {
			return ErrAmountOfFeeOutOfRange
		}
	}
	return nil
}

func calcMinFeeAmount(amount uint64) uint64 {
	return uint64(math.Ceil(float64(amount) / 1000))
}

func calcMaxFeeAmount(amount uint64, maxFeeRate float64) uint64 {
	return uint64(math.Ceil(float64(amount) * maxFeeRate))
}
