package match

import (
	"math"

	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
)

var (
	// ErrAmountOfFeeExceedMaximum represent The fee charged is exceeded the maximum
	ErrAmountOfFeeExceedMaximum = errors.New("amount of fee greater than max fee amount")
	// ErrFeeMoreThanOneAsset represent the fee charged can only have one asset
	ErrFeeMoreThanOneAsset      = errors.New("fee can only be an asset")
)

// FeeStrategy used to indicate how to charge a matching fee
type FeeStrategy interface {
	// Allocate will allocate the price differential in matching transaction to the participants and the fee
	// @param receiveAmounts the amount of assets that the participants in the matching transaction can received when no fee is considered
	// @param priceDiff price differential of matching transaction
	// @return the amount of assets that the participants in the matching transaction can received when fee is considered
	// @return the amount of assets returned to the transaction participant when the fee exceeds a certain ratio
	// @return the amount of fees
	Allocate(receiveAmounts []*bc.AssetAmount, priceDiff *bc.AssetAmount) ([]*bc.AssetAmount, []*bc.AssetAmount, []*bc.AssetAmount)

	// Validate verify that the fee charged for a matching transaction is correct
	Validate(receiveAmounts []*bc.AssetAmount, priceDiff *bc.AssetAmount, feeAmounts map[bc.AssetID]int64) error
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
func (d *DefaultFeeStrategy) Allocate(receiveAmounts []*bc.AssetAmount, priceDiff *bc.AssetAmount) ([]*bc.AssetAmount, []*bc.AssetAmount, []*bc.AssetAmount) {
	receivedAfterDeductFee := make([]*bc.AssetAmount, len(receiveAmounts))
	copy(receivedAfterDeductFee, receiveAmounts)

	if priceDiff.Amount == 0 {
		return receivedAfterDeductFee, nil, nil
	}

	var maxFeeAmount int64
	for _, receiveAmount := range receiveAmounts {
		if *receiveAmount.AssetId == *priceDiff.AssetId {
			maxFeeAmount = calcMaxFeeAmount(receiveAmount.Amount, d.maxFeeRate)
		}
	}

	priceDiffAmount := int64(priceDiff.Amount)
	feeAmount, reminder := priceDiffAmount, int64(0)
	if priceDiffAmount > maxFeeAmount {
		feeAmount = maxFeeAmount
		reminder = priceDiffAmount - maxFeeAmount
	}

	// There is the remaining amount after paying the handling fee, assign it evenly to participants in the transaction
	averageAmount := reminder / int64(len(receiveAmounts))
	if averageAmount == 0 {
		averageAmount = 1
	}

	var refundAmounts []*bc.AssetAmount
	for i := 0; i < len(receiveAmounts) && reminder > 0; i++ {
		amount := averageAmount
		if i == len(receiveAmounts)-1 {
			amount = reminder
		}
		refundAmounts = append(refundAmounts, &bc.AssetAmount{AssetId: priceDiff.AssetId, Amount: uint64(amount)})
		reminder -= averageAmount
	}

	feeAmounts := []*bc.AssetAmount{{AssetId: priceDiff.AssetId, Amount: uint64(feeAmount)}}
	return receivedAfterDeductFee, refundAmounts, feeAmounts
}

// Validate verify that the fee charged for a matching transaction is correct
func (d *DefaultFeeStrategy) Validate(receiveAmounts []*bc.AssetAmount, priceDiff *bc.AssetAmount, feeAmounts map[bc.AssetID]int64) error {
	if len(feeAmounts) > 1 {
		return ErrFeeMoreThanOneAsset
	}

	for _, receiveAmount := range receiveAmounts {
		if feeAmount, ok := feeAmounts[*receiveAmount.AssetId]; ok {
			if feeAmount > calcMaxFeeAmount(receiveAmount.Amount, d.maxFeeRate) {
				return ErrAmountOfFeeExceedMaximum
			}
		}
	}
	return nil
}

func calcMaxFeeAmount(amount uint64, maxFeeRate float64) int64 {
	return int64(math.Ceil(float64(amount) * maxFeeRate))
}
