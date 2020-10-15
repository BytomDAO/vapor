package match

import (
	"math"

	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
)

var (
	// ErrInvalidAmountOfFee represent The fee charged is invalid
	ErrInvalidAmountOfFee = errors.New("amount of fee is invalid")
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
	// @return reallocated assets after calculating fees
	Allocate(receiveAmounts, priceDiffs []*bc.AssetAmount, isMakers []bool) *AllocatedAssets

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
func (d *DefaultFeeStrategy) Allocate(receiveAmounts, priceDiffs []*bc.AssetAmount, isMakers []bool) *AllocatedAssets {
	receives := make([]*bc.AssetAmount, len(receiveAmounts))
	fees := make([]*bc.AssetAmount, len(receiveAmounts))

	for i, receiveAmount := range receiveAmounts {
		fee := d.calcFeeAmount(receiveAmount.Amount)
		receives[i] = &bc.AssetAmount{AssetId: receiveAmount.AssetId, Amount: receiveAmount.Amount - fee}
		fees[i] = &bc.AssetAmount{AssetId: receiveAmount.AssetId, Amount: fee}

		if !isMakers[i] {
			for _, priceDiff := range priceDiffs {
				if *priceDiff.AssetId == *receiveAmount.AssetId {
					fee = d.calcFeeAmount(priceDiff.Amount)
					priceDiff.Amount -= fee
					fees[i].Amount += fee
				}
			}
		}
	}
	return &AllocatedAssets{Receives: receives, Fees: fees}
}
// Validate verify that the fee charged for a matching transaction is correct
func (d *DefaultFeeStrategy) Validate(receiveAmounts []*bc.AssetAmount, feeAmounts map[bc.AssetID]uint64) error {
	for _, receiveAmount := range receiveAmounts {
		realFeeAmount := feeAmounts[*receiveAmount.AssetId]
		feeAmount := d.calcFeeAmount(receiveAmount.Amount)
		if realFeeAmount < feeAmount {
			return ErrInvalidAmountOfFee
		}
	}
	return nil
}

func (d *DefaultFeeStrategy) calcFeeAmount(amount uint64) uint64 {
	return uint64(math.Ceil(float64(amount) / 1000))
}
