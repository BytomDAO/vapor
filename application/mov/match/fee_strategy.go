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
	Fees     []*bc.AssetAmount
}

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
type DefaultFeeStrategy struct{}

// NewDefaultFeeStrategy return a new instance of DefaultFeeStrategy
func NewDefaultFeeStrategy() *DefaultFeeStrategy {
	return &DefaultFeeStrategy{}
}

// Allocate will allocate the price differential in matching transaction to the participants and the fee
func (d *DefaultFeeStrategy) Allocate(receiveAmounts, priceDiffs []*bc.AssetAmount) *AllocatedAssets {
	receives := make([]*bc.AssetAmount, len(receiveAmounts))
	fees := make([]*bc.AssetAmount, len(receiveAmounts))

	for i, receiveAmount := range receiveAmounts {
		standFee := d.calcMinFeeAmount(receiveAmount.Amount)
		fee := standFee + priceDiffs[i].Amount
		if maxFeeAmount := d.calcMaxFeeAmount(receiveAmount.Amount); fee > maxFeeAmount {
			fee = maxFeeAmount
		}

		receives[i] = &bc.AssetAmount{AssetId: receiveAmount.AssetId, Amount: receiveAmount.Amount - standFee}
		fees[i] = &bc.AssetAmount{AssetId: receiveAmount.AssetId, Amount: fee}
	}
	return &AllocatedAssets{Receives: receives, Fees: fees}
}

// Validate verify that the fee charged for a matching transaction is correct
func (d *DefaultFeeStrategy) Validate(receiveAmounts []*bc.AssetAmount, feeAmounts map[bc.AssetID]uint64) error {
	for _, receiveAmount := range receiveAmounts {
		feeAmount := feeAmounts[*receiveAmount.AssetId]
		maxFeeAmount := d.calcMaxFeeAmount(receiveAmount.Amount)
		minFeeAmount := d.calcMinFeeAmount(receiveAmount.Amount)
		if feeAmount < minFeeAmount || feeAmount > maxFeeAmount {
			return ErrAmountOfFeeOutOfRange
		}
	}
	return nil
}

func (d *DefaultFeeStrategy) calcMinFeeAmount(amount uint64) uint64 {
	return uint64(math.Ceil(float64(amount) / 1000))
}

func (d *DefaultFeeStrategy) calcMaxFeeAmount(amount uint64) uint64 {
	return uint64(math.Ceil(float64(amount) * 0.05))
}
