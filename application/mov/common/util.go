package common

import (
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

func OutputToOrder(tx *types.Tx, outputIndex int) (*Order, error) {
	outputID := tx.OutputID(outputIndex)
	entry, err := tx.Entry(*outputID)
	if err != nil {
		return nil, err
	}

	output, ok := entry.(*bc.IntraChainOutput)
	if !ok {
		return nil, errors.New("output is not type of intra chain output")
	}

	contractArgs := DecodeDexProgram(tx.Outputs[outputIndex].ControlProgram())
	assetAmount := tx.Outputs[outputIndex].AssetAmount()
	return &Order{
		FromAssetID: assetAmount.AssetId,
		ToAssetID:   &contractArgs.RequestedAsset,
		Rate:        float64(contractArgs.RatioMolecule) / float64(contractArgs.RatioDenominator),
		Utxo: &MovUtxo{
			SourceID:       output.Source.Ref,
			Amount:         assetAmount.Amount,
			SourcePos:      uint64(outputIndex),
			ControlProgram: output.ControlProgram.Code,
		},
	}, nil
}

func InputToOrder(txInput *types.TxInput) (*Order, error) {
	input, ok := txInput.TypedInput.(*types.SpendInput)
	if !ok {
		return nil, errors.New("input is not type of spend input")
	}

	contractArgs := DecodeDexProgram(input.ControlProgram)
	return &Order{
		FromAssetID: input.AssetId,
		ToAssetID:   &contractArgs.RequestedAsset,
		Rate:        float64(contractArgs.RatioMolecule) / float64(contractArgs.RatioDenominator),
		Utxo: &MovUtxo{
			SourceID:       &input.SourceID,
			Amount:         input.Amount,
			SourcePos:     	input.SourcePosition,
			ControlProgram: input.ControlProgram,
		},
	}, nil
}


// ------------- mock -------------------

type DexContractArgs struct {
	RequestedAsset   bc.AssetID
	RatioMolecule    uint64
	RatioDenominator uint64
	SellerProgram    []byte
	SellerKey        []byte
}

func DecodeDexProgram(program []byte) *DexContractArgs {
	return nil
}
