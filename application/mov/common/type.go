package common

import (
	"fmt"

	"github.com/vapor/consensus/segwit"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

type MovUtxo struct {
	SourceID       *bc.Hash
	SourcePos      uint64
	Amount         uint64
	ControlProgram []byte
}

type Order struct {
	FromAssetID *bc.AssetID
	ToAssetID   *bc.AssetID
	Utxo        *MovUtxo
	Rate        float64
}

type OrderSlice []*Order

func (o OrderSlice) Len() int {
	return len(o)
}
func (o OrderSlice) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}
func (o OrderSlice) Less(i, j int) bool {
	return o[i].Rate < o[j].Rate
}

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
			SourcePos:     	input.SourcePosition,
			ControlProgram: input.ControlProgram,
		},
	}, nil
}

func (o *Order) GetTradePair() *TradePair {
	return &TradePair{FromAssetID: o.FromAssetID, ToAssetID: o.ToAssetID}
}

func (o *Order) Key() string {
	return fmt.Sprintf("%s:%d", o.Utxo.SourceID, o.Utxo.SourcePos)
}

type TradePair struct {
	FromAssetID *bc.AssetID
	ToAssetID   *bc.AssetID
	Count       int
}

func (t *TradePair) Reverse() *TradePair {
	return &TradePair{
		FromAssetID: t.ToAssetID,
		ToAssetID:   t.FromAssetID,
	}
}

func (t *TradePair) Key() string {
	return fmt.Sprintf("%s:%s", t.FromAssetID, t.ToAssetID)
}

type MovDatabaseState struct {
	Height uint64
	Hash   *bc.Hash
}
