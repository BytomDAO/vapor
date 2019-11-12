package common

import (
	"encoding/hex"
	"fmt"

	"github.com/vapor/consensus/segwit"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
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
	FromAssetID *bc.AssetID
	ToAssetID   *bc.AssetID
	Utxo        *MovUtxo
	Rate        float64
}

// OrderSlice is define for order's sort
type OrderSlice []*Order

func (o OrderSlice) Len() int      { return len(o) }
func (o OrderSlice) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o OrderSlice) Less(i, j int) bool {
	if o[i].Rate == o[j].Rate {
		return hex.EncodeToString(o[i].UTXOHash().Bytes()) < hex.EncodeToString(o[j].UTXOHash().Bytes())
	}
	return o[i].Rate < o[j].Rate
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
		FromAssetID: assetAmount.AssetId,
		ToAssetID:   &contractArgs.RequestedAsset,
		Rate:        float64(contractArgs.RatioNumerator) / float64(contractArgs.RatioDenominator),
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
		FromAssetID: input.AssetId,
		ToAssetID:   &contractArgs.RequestedAsset,
		Rate:        float64(contractArgs.RatioNumerator) / float64(contractArgs.RatioDenominator),
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
