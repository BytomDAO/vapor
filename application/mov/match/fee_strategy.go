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

const forkBlockHeightAt20201028 = 78968116

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
		fee := calcMinFeeAmount(receiveAmount.Amount)
		receives[i] = &bc.AssetAmount{AssetId: receiveAmount.AssetId, Amount: receiveAmount.Amount - fee}
		fees[i] = &bc.AssetAmount{AssetId: receiveAmount.AssetId, Amount: fee}

		if i == takerPos {
			for _, priceDiff := range priceDiffs {
				if *priceDiff.AssetId == *receiveAmount.AssetId {
					fee = calcMinFeeAmount(priceDiff.Amount)
					priceDiff.Amount -= fee
					fees[i].Amount += fee
				}
			}
		}
	}
	return &AllocatedAssets{Receives: receives, Fees: fees}
}

// Validate verify that the fee charged for a matching transaction is correct
func (d *DefaultFeeStrategy) Validate(receiveAmounts, priceDiffs []*bc.AssetAmount, feeAmounts map[bc.AssetID]uint64, blockHeight uint64) error {
	if blockHeight < forkBlockHeightAt20201028 {
		return legendValidateFee(receiveAmounts, feeAmounts)
	}
	return validateFee(receiveAmounts, priceDiffs, feeAmounts)
}

func validateFee(receiveAmounts, priceDiffs []*bc.AssetAmount, feeAmounts map[bc.AssetID]uint64) error {
	existTaker := false
	for _, receiveAmount := range receiveAmounts {
		feeAmount := calcMinFeeAmount(receiveAmount.Amount)
		realFeeAmount := feeAmounts[*receiveAmount.AssetId]
		if equalsFeeAmount(realFeeAmount, feeAmount) {
			continue
		}

		if existTaker {
			return ErrInvalidAmountOfFee
		}

		for _, priceDiff := range priceDiffs {
			if *priceDiff.AssetId == *receiveAmount.AssetId {
				feeAmount += calcMinFeeAmount(priceDiff.Amount)
			}
		}

		if !equalsFeeAmount(realFeeAmount, feeAmount) {
			return ErrInvalidAmountOfFee
		}
		existTaker = true
	}
	return nil
}

func equalsFeeAmount(realFeeAmount, feeAmount uint64) bool {
	var tolerance float64 = 5
	return math.Abs(float64(realFeeAmount)-float64(feeAmount)) < tolerance
}

func legendValidateFee(receiveAmounts []*bc.AssetAmount, feeAmounts map[bc.AssetID]uint64) error {
	for _, receiveAmount := range receiveAmounts {
		realFeeAmount := feeAmounts[*receiveAmount.AssetId]
		minFeeAmount := calcMinFeeAmount(receiveAmount.Amount)
		maxFeeAmount := calcMaxFeeAmount(receiveAmount.Amount)
		if realFeeAmount < minFeeAmount || realFeeAmount > maxFeeAmount {
			return ErrInvalidAmountOfFee
		}
	}
	return nil
}

func calcMinFeeAmount(amount uint64) uint64 {
	return uint64(math.Ceil(float64(amount) / 1000))
}

func calcMaxFeeAmount(amount uint64) uint64 {
	return uint64(math.Ceil(float64(amount) * 0.05))
}
