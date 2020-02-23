package common

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/bytom/vapor/consensus/segwit"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
)

// MovUtxo store the utxo information for mov order
type MovUtxo struct {
	SourceID       *bc.Hash
	SourcePos      uint64
	Amount         uint64
	ControlProgram []byte
}

// Order store all the order information
type Order struct {
	FromAssetID      *bc.AssetID
	ToAssetID        *bc.AssetID
	Utxo             *MovUtxo
	RatioNumerator   int64
	RatioDenominator int64
}

// Rate return the exchange represented by float64
func (o *Order) Rate() float64 {
	if o.RatioDenominator == 0 {
		return 0
	}
	rate := big.NewRat(o.RatioNumerator, o.RatioDenominator)
	result, _ := rate.Float64()
	return result
}

// cmpRate compares rate of x and y and returns -1 if x <  y, 0 if x == y, +1 if x >  y
func (o *Order) cmpRate(other *Order) int {
	rate := big.NewRat(o.RatioNumerator, o.RatioDenominator)
	otherRate := big.NewRat(other.RatioNumerator, other.RatioDenominator)
	return rate.Cmp(otherRate)
}

// Cmp first compare the rate, if rate is equals, then compare the utxo hash
func (o *Order) Cmp(other *Order) int {
	cmp := o.cmpRate(other)
	if cmp == 0 {
		if hex.EncodeToString(o.UTXOHash().Bytes()) < hex.EncodeToString(other.UTXOHash().Bytes()) {
			return -1
		}
		return 1
	}
	return cmp
}

// OrderSlice is define for order's sort
type OrderSlice []*Order

func (o OrderSlice) Len() int      { return len(o) }
func (o OrderSlice) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o OrderSlice) Less(i, j int) bool {
	return o[i].Cmp(o[j]) < 0
}

// NewOrderFromOutput convert txinput to order
func NewOrderFromOutput(tx *types.Tx, outputIndex int) (*Order, error) {
	outputID := tx.OutputID(outputIndex)
	output, err := tx.IntraChainOutput(*outputID)
	if err != nil {
		return nil, err
	}

	contractArgs, err := segwit.DecodeP2WMCProgram(output.ControlProgram.Code)
	if err != nil {
		return nil, err
	}

	assetAmount := output.Source.Value
	return &Order{
		FromAssetID:      assetAmount.AssetId,
		ToAssetID:        &contractArgs.RequestedAsset,
		RatioNumerator:   contractArgs.RatioNumerator,
		RatioDenominator: contractArgs.RatioDenominator,
		Utxo: &MovUtxo{
			SourceID:       output.Source.Ref,
			Amount:         assetAmount.Amount,
			SourcePos:      uint64(outputIndex),
			ControlProgram: output.ControlProgram.Code,
		},
	}, nil
}

// NewOrderFromInput convert txoutput to order
func NewOrderFromInput(tx *types.Tx, inputIndex int) (*Order, error) {
	input, ok := tx.Inputs[inputIndex].TypedInput.(*types.SpendInput)
	if !ok {
		return nil, errors.New("input is not type of spend input")
	}

	contractArgs, err := segwit.DecodeP2WMCProgram(input.ControlProgram)
	if err != nil {
		return nil, err
	}

	return &Order{
		FromAssetID:      input.AssetId,
		ToAssetID:        &contractArgs.RequestedAsset,
		RatioNumerator:   contractArgs.RatioNumerator,
		RatioDenominator: contractArgs.RatioDenominator,
		Utxo: &MovUtxo{
			SourceID:       &input.SourceID,
			Amount:         input.Amount,
			SourcePos:      input.SourcePosition,
			ControlProgram: input.ControlProgram,
		},
	}, nil
}

// Key return the unique key for representing this order
func (o *Order) Key() string {
	return fmt.Sprintf("%s:%d", o.Utxo.SourceID, o.Utxo.SourcePos)
}

// TradePair return the trade pair info
func (o *Order) TradePair() *TradePair {
	return &TradePair{FromAssetID: o.FromAssetID, ToAssetID: o.ToAssetID}
}

// UTXOHash calculate the utxo hash of this order
func (o *Order) UTXOHash() *bc.Hash {
	prog := &bc.Program{VmVersion: 1, Code: o.Utxo.ControlProgram}
	src := &bc.ValueSource{
		Ref:      o.Utxo.SourceID,
		Value:    &bc.AssetAmount{AssetId: o.FromAssetID, Amount: o.Utxo.Amount},
		Position: o.Utxo.SourcePos,
	}
	hash := bc.EntryID(bc.NewIntraChainOutput(src, prog, 0))
	return &hash
}

// TradePair is the object for record trade pair info
type TradePair struct {
	FromAssetID *bc.AssetID
	ToAssetID   *bc.AssetID
	Count       int
}

// Key return the unique key for representing this trade pair
func (t *TradePair) Key() string {
	return fmt.Sprintf("%s:%s", t.FromAssetID, t.ToAssetID)
}

// Reverse return the reverse trade pair object
func (t *TradePair) Reverse() *TradePair {
	return &TradePair{
		FromAssetID: t.ToAssetID,
		ToAssetID:   t.FromAssetID,
	}
}

// MovDatabaseState is object to record DB image status
type MovDatabaseState struct {
	Height uint64
	Hash   *bc.Hash
}
