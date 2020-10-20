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
	Allocate(receiveAmounts, priceDiffs []*bc.AssetAmount, takerPos int) *AllocatedAssets

	// Validate verify that the fee charged for a matching transaction is correct
	Validate(receiveAmounts, priceDiffs []*bc.AssetAmount, feeAmounts map[bc.AssetID]uint64, blockHeight uint64) error
}

// DefaultFeeStrategy represent the default fee charge strategy
type DefaultFeeStrategy struct{}

// NewDefaultFeeStrategy return a new instance of DefaultFeeStrategy
func NewDefaultFeeStrategy() *DefaultFeeStrategy {
	return &DefaultFeeStrategy{}
}

// Allocate will allocate the price differential in matching transaction to the participants and the fee
func (d *DefaultFeeStrategy) Allocate(receiveAmounts, priceDiffs []*bc.AssetAmount, takerPos int) *AllocatedAssets {
	receives := make([]*bc.AssetAmount, len(receiveAmounts))
	fees := make([]*bc.AssetAmount, len(receiveAmounts))

	for i, receiveAmount := range receiveAmounts {
		fee := d.calcMinFeeAmount(receiveAmount.Amount)
		receives[i] = &bc.AssetAmount{AssetId: receiveAmount.AssetId, Amount: receiveAmount.Amount - fee}
		fees[i] = &bc.AssetAmount{AssetId: receiveAmount.AssetId, Amount: fee}

		if i == takerPos {
			for _, priceDiff := range priceDiffs {
				if *priceDiff.AssetId == *receiveAmount.AssetId {
					fee = d.calcMinFeeAmount(priceDiff.Amount)
					priceDiff.Amount -= fee
					fees[i].Amount += fee
				}
			}
		}
	}
	return &AllocatedAssets{Receives: receives, Fees: fees}
}

const forkBlockHeight = 83000000

// Validate verify that the fee charged for a matching transaction is correct
func (d *DefaultFeeStrategy) Validate(receiveAmounts, priceDiffs []*bc.AssetAmount, feeAmounts map[bc.AssetID]uint64, blockHeight uint64) error {
	existTaker := false
	for _, receiveAmount := range receiveAmounts {
		realFeeAmount := feeAmounts[*receiveAmount.AssetId]
		minFeeAmount := d.calcMinFeeAmount(receiveAmount.Amount)
		if blockHeight <= forkBlockHeight {
			maxFeeAmount := d.calcMaxFeeAmount(receiveAmount.Amount)
			if realFeeAmount < minFeeAmount || realFeeAmount > maxFeeAmount {
				return ErrInvalidAmountOfFee
			}
		} else {
			if realFeeAmount != minFeeAmount && existTaker {
				return ErrInvalidAmountOfFee
			}

			for _, priceDiff := range priceDiffs {
				if priceDiff.AssetId == receiveAmount.AssetId {
					minFeeAmount += d.calcMinFeeAmount(priceDiff.Amount)
				}
			}

			if realFeeAmount != minFeeAmount {
				return ErrInvalidAmountOfFee
			}
			existTaker = true
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
