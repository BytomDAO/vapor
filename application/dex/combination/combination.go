package combination

import (
	"encoding/hex"
	"errors"

	"github.com/vapor/application/dex/common"
	vprCommon "github.com/vapor/common"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	buyOrderOutputIndex  = 1
	sellOrderOutputIndex = 2
)

type combination struct {}

// GenerateCombinationTxs combine two opposite pending orders.
// for example, the ordersX want change A with B, then the ordersY must change B with A.
// the input order's rate must in descending order.
func (c *combination) GenerateCombinationTxs(ordersX, ordersY []*common.Order) ([]*types.Tx, error) {
	buyOrders := vprCommon.NewStack()
	for i := len(ordersX) - 1; i >= 0; i-- {
		buyOrders.Push(ordersX[i])
	}

	sellOrders := vprCommon.NewStack()
	for i := len(ordersY) - 1; i >= 0; i-- {
		sellOrders.Push(ordersY[i])
	}

	combinationTxs := []*types.Tx{}
	for buyOrders.Len() > 0 && sellOrders.Len() > 0 {
		buyOrder := buyOrders.Peek().(*common.Order)
		sellOrder := sellOrders.Peek().(*common.Order)
		if canBeCombined(buyOrder, sellOrder) {
			tx, err := buildCombinationTx(buyOrder, sellOrder)
			if err != nil {
				return nil, err
			}

			combinationTxs = append(combinationTxs, tx)
			if err := adjustOrderTable(tx, buyOrders, sellOrders); err != nil {
				return nil, err
			}
		} else {
			break
		}
	}
	return combinationTxs, nil
}

func canBeCombined(buyOrder, sellOrder *common.Order) bool {
	if buyOrder.ToAssetID != sellOrder.FromAssetID || sellOrder.ToAssetID != buyOrder.FromAssetID {
		return false
	}
	return 1/buyOrder.Rate >= sellOrder.Rate
}

func buildCombinationTx(buyOrder, sellOrder *common.Order) (*types.Tx, error) {
	txData := types.TxData{}
	txData.Inputs = append(txData.Inputs, types.NewSpendInput(nil, *buyOrder.Utxo.SourceID, *buyOrder.FromAssetID, buyOrder.Utxo.Amount, buyOrder.Utxo.SourcePos, buyOrder.Utxo.ControlProgram))
	txData.Inputs = append(txData.Inputs, types.NewSpendInput(nil, *sellOrder.Utxo.SourceID, *sellOrder.FromAssetID, sellOrder.Utxo.Amount, sellOrder.Utxo.SourcePos, sellOrder.Utxo.ControlProgram))

	buyContractArgs := DecodeDexProgram(buyOrder.Utxo.ControlProgram)
	buyRequestAmount := calcToAmountByFromAmount(buyOrder.Utxo.Amount, buyContractArgs)
	buyReceiveAmount := min(buyRequestAmount, sellOrder.Utxo.Amount)
	txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*buyOrder.ToAssetID, buyReceiveAmount, buyContractArgs.SellerProgram))

	buyShouldPayAmount := calcFromAmountByToAmount(buyReceiveAmount, buyContractArgs)
	if buyOrder.Utxo.Amount > buyShouldPayAmount {
		txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*buyOrder.FromAssetID, buyOrder.Utxo.Amount-buyShouldPayAmount, buyOrder.Utxo.ControlProgram))
	}

	sellContractArgs := DecodeDexProgram(sellOrder.Utxo.ControlProgram)
	sellRequestAmount := calcToAmountByFromAmount(sellOrder.Utxo.Amount, sellContractArgs)
	sellReceiveAmount := min(sellRequestAmount, buyOrder.Utxo.Amount)
	txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*sellOrder.ToAssetID, sellReceiveAmount, sellContractArgs.SellerProgram))

	sellShouldPayAmount := calcFromAmountByToAmount(sellReceiveAmount, sellContractArgs)
	if sellOrder.Utxo.Amount > sellShouldPayAmount {
		txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*sellOrder.FromAssetID, sellOrder.Utxo.Amount-sellShouldPayAmount, sellOrder.Utxo.ControlProgram))
	}

	// fee output
	if buyShouldPayAmount > sellReceiveAmount {
		txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*sellOrder.ToAssetID, buyShouldPayAmount-sellRequestAmount, []byte{ /** node address */ }))
	}

	if sellShouldPayAmount > buyReceiveAmount {
		txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*buyOrder.ToAssetID, sellShouldPayAmount-buyReceiveAmount, []byte{ /** node address */ }))
	}
	return types.NewTx(txData), nil
}

func adjustOrderTable(tx *types.Tx, buyOrders, sellOrders *vprCommon.Stack) error {
	buyOrder := buyOrders.Pop().(*common.Order)
	sellOrder := sellOrders.Pop().(*common.Order)

	if hex.EncodeToString(tx.Outputs[buyOrderOutputIndex].ControlProgram()) == hex.EncodeToString(tx.Inputs[0].ControlProgram()) {
		utxo, err := outputToUTXO(tx, buyOrderOutputIndex)
		if err != nil {
			return err
		}

		buyOrders.Push(&common.Order{FromAssetID: buyOrder.FromAssetID, ToAssetID: buyOrder.ToAssetID, Rate: buyOrder.Rate, Utxo: utxo})
		return nil
	}

	if hex.EncodeToString(tx.Outputs[sellOrderOutputIndex].ControlProgram()) == hex.EncodeToString(tx.Inputs[1].ControlProgram()) {
		utxo, err := outputToUTXO(tx, sellOrderOutputIndex)
		if err != nil {
			return err
		}

		sellOrders.Push(&common.Order{FromAssetID: sellOrder.FromAssetID, ToAssetID: sellOrder.ToAssetID, Rate: sellOrder.Rate, Utxo: utxo})
	}
	return nil
}

func outputToUTXO(tx *types.Tx, outputIndex int) (*common.DexUtxo, error) {
	outputID := tx.OutputID(outputIndex)
	entry, err := tx.Entry(*outputID)
	if err != nil {
		return nil, err
	}

	output, ok := entry.(*bc.IntraChainOutput)
	if !ok {
		return nil, errors.New("output is not type of intra chain output")
	}

	assetAmount := tx.Outputs[outputIndex].AssetAmount()
	return &common.DexUtxo{
		SourceID:       output.Source.Ref,
		Amount:         assetAmount.Amount,
		SourcePos:      uint64(outputIndex),
		ControlProgram: output.ControlProgram.Code,
	}, nil
}

func calcToAmountByFromAmount(fromAmount uint64, contractArg *DexContractArgs) uint64 {
	return fromAmount * contractArg.RatioMolecule / contractArg.RatioDenominator
}

func calcFromAmountByToAmount(toAmount uint64, contractArg *DexContractArgs) uint64 {
	return toAmount * contractArg.RatioDenominator / contractArg.RatioMolecule
}

func min(x, y uint64) uint64 {
	if x < y {
		return x
	}
	return y
}

// ------------- mock -------------------

type DexContractArgs struct {
	RequestedAsset   bc.Hash
	RatioMolecule    uint64
	RatioDenominator uint64
	SellerProgram    []byte
	SellerKey        []byte
}

func DecodeDexProgram(program []byte) *DexContractArgs {
	return nil
}
