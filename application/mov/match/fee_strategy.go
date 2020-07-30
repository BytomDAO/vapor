package match

import (
	"math"

	"github.com/bytom/vapor/consensus/segwit"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
)

var (
	// ErrInvalidAmountOfFee represent The fee charged is invalid
	ErrInvalidAmountOfFee = errors.New("amount of fee is invalid")
)

const (
	// MakerFeeRate represent the fee rate of maker, which in units of 10000
	MakerFeeRate int64 = 0
	// TakerFeeRate represent the fee rate of taker
	TakerFeeRate int64 = 5
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
	// @param priceDiffs price differential of matching transaction, it will be refunded to the taker
	// @return reallocated assets after calculating fees
	Allocate(receiveAmounts, priceDiffs []*bc.AssetAmount, makerFlags []MakerFlag) *AllocatedAssets

	// Validate verify that the fee charged for a matching transaction is correct
	Validate(receiveAmounts, priceDiffs []*bc.AssetAmount, feeAmounts map[bc.AssetID]uint64, makerFlags []MakerFlag) error
}

// DefaultFeeStrategy represent the default fee charge strategy
type DefaultFeeStrategy struct{}

// NewDefaultFeeStrategy return a new instance of DefaultFeeStrategy
func NewDefaultFeeStrategy() *DefaultFeeStrategy {
	return &DefaultFeeStrategy{}
}

// Allocate will allocate the price differential in matching transaction to the participants and the fee
func (d *DefaultFeeStrategy) Allocate(receiveAmounts, priceDiffs []*bc.AssetAmount, makerFlags []MakerFlag) *AllocatedAssets {
	receives := make([]*bc.AssetAmount, len(receiveAmounts))
	fees := make([]*bc.AssetAmount, len(receiveAmounts))

	for i, receiveAmount := range receiveAmounts {
		makerFlag := makerFlags[i]
		fee := calcFeeAmount(receiveAmount.Amount, makerFlag.IsMaker)
		if makerFlag.ContractVersion == segwit.MagneticV1 {
			fee = legendCalcMinFeeAmount(receiveAmount.Amount)
		}

		receives[i] = &bc.AssetAmount{AssetId: receiveAmount.AssetId, Amount: receiveAmount.Amount - fee}
		fees[i] = &bc.AssetAmount{AssetId: receiveAmount.AssetId, Amount: fee}

		if makerFlag.ContractVersion == segwit.MagneticV2 && !makerFlag.IsMaker {
			for _, priceDiff := range priceDiffs {
				if *priceDiff.AssetId == *receiveAmount.AssetId {
					fee = calcFeeAmount(priceDiff.Amount, makerFlag.IsMaker)
					priceDiff.Amount -= fee
					fees[i].Amount += fee
				}
			}
		}
	}
	return &AllocatedAssets{Receives: receives, Fees: fees}
}

// Validate verify that the fee charged for a matching transaction is correct
func (d *DefaultFeeStrategy) Validate(receiveAmounts, priceDiffs []*bc.AssetAmount, feeAmounts map[bc.AssetID]uint64, makerFlags []MakerFlag) error {
	for i, receiveAmount := range receiveAmounts {
		receiveAssetID := receiveAmount.AssetId
		feeAmount := feeAmounts[*receiveAssetID]

		if makerFlags[i].ContractVersion == segwit.MagneticV1 {
			return legendValidate(receiveAmount, feeAmount)
		}

		expectFee := calcFeeAmount(receiveAmount.Amount, makerFlags[i].IsMaker)
		if !makerFlags[i].IsMaker {
			for _, priceDiff := range priceDiffs {
				if *priceDiff.AssetId == *receiveAssetID {
					expectFee += calcFeeAmount(priceDiff.Amount, false)
				}
			}
		}

		if feeAmount != expectFee {
			return ErrInvalidAmountOfFee
		}
	}
	return nil
}

func calcFeeAmount(amount uint64, isMaker bool) uint64 {
	feeRate := TakerFeeRate
	if isMaker {
		feeRate = MakerFeeRate
	}
	return uint64(math.Ceil(float64(amount) * float64(feeRate) / 1E4))
}

func legendValidate(receiveAmount *bc.AssetAmount, feeAmount uint64) error {
	maxFeeAmount := legendCalcMaxFeeAmount(receiveAmount.Amount)
	minFeeAmount := legendCalcMinFeeAmount(receiveAmount.Amount)
	if feeAmount < minFeeAmount || feeAmount > maxFeeAmount {
		return ErrInvalidAmountOfFee
	}
	return nil
}

func legendCalcMinFeeAmount(amount uint64) uint64 {
	return uint64(math.Ceil(float64(amount) / 1000))
}

func legendCalcMaxFeeAmount(amount uint64) uint64 {
	return uint64(math.Ceil(float64(amount) * 0.05))
}
