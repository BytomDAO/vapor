package match

import (
	"math/big"
	"encoding/hex"
	"errors"

	"github.com/vapor/application/dex/common"
	vprCommon "github.com/vapor/common"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm"
)

const (
	buyOrderOutputIndex  = 1
	sellOrderOutputIndex = 2
)

type MatchEngine struct{}

type OrderTable struct {
	BuyOrders  []*common.Order
	SellOrders []*common.Order
}

// GenerateMatchedTxs match two opposite pending orders.
// for example, the buy orders want change A with B, then the sell orders must change B with A.
// the input order's rate must in descending order.
func (c *MatchEngine) GenerateMatchedTxs(orderTable *OrderTable) ([]*types.Tx, error) {
	buyOrders := vprCommon.NewStack()
	for i := len(orderTable.BuyOrders) - 1; i >= 0; i-- {
		buyOrders.Push(orderTable.BuyOrders[i])
	}

	sellOrders := vprCommon.NewStack()
	for i := len(orderTable.SellOrders) - 1; i >= 0; i-- {
		sellOrders.Push(orderTable.SellOrders[i])
	}

	matchedTxs := []*types.Tx{}
	for buyOrders.Len() > 0 && sellOrders.Len() > 0 {
		buyOrder := buyOrders.Peek().(*common.Order)
		sellOrder := sellOrders.Peek().(*common.Order)
		if canBeMatched(buyOrder, sellOrder) {
			tx, err := buildMatchTx(buyOrder, sellOrder)
			if err != nil {
				return nil, err
			}

			matchedTxs = append(matchedTxs, tx)
			if err := adjustOrderTable(tx, buyOrders, sellOrders); err != nil {
				return nil, err
			}
		} else {
			break
		}
	}
	return matchedTxs, nil
}

func canBeMatched(buyOrder, sellOrder *common.Order) bool {
	if buyOrder.ToAssetID != sellOrder.FromAssetID || sellOrder.ToAssetID != buyOrder.FromAssetID {
		return false
	}

	buyContractArgs := DecodeDexProgram(buyOrder.Utxo.ControlProgram)
	sellContractArgs := DecodeDexProgram(sellOrder.Utxo.ControlProgram)
	buyRate := big.NewFloat(0).Quo(big.NewFloat(0).SetUint64(buyContractArgs.RatioDenominator), big.NewFloat(0).SetUint64(buyContractArgs.RatioMolecule))
	sellRate := big.NewFloat(0).Quo(big.NewFloat(0).SetUint64(sellContractArgs.RatioMolecule), big.NewFloat(0).SetUint64(buyContractArgs.RatioDenominator))
	return buyRate.Cmp(sellRate) >= 0
}

func buildMatchTx(buyOrder, sellOrder *common.Order) (*types.Tx, error) {
	txData := types.TxData{}
	txData.Inputs = append(txData.Inputs, types.NewSpendInput(nil, *buyOrder.Utxo.SourceID, *buyOrder.FromAssetID, buyOrder.Utxo.Amount, buyOrder.Utxo.SourcePos, buyOrder.Utxo.ControlProgram))
	txData.Inputs = append(txData.Inputs, types.NewSpendInput(nil, *sellOrder.Utxo.SourceID, *sellOrder.FromAssetID, sellOrder.Utxo.Amount, sellOrder.Utxo.SourcePos, sellOrder.Utxo.ControlProgram))

	buyContractArgs := DecodeDexProgram(buyOrder.Utxo.ControlProgram)
	buyRequestAmount := calcToAmountByFromAmount(buyOrder.Utxo.Amount, buyContractArgs)
	buyReceiveAmount := min(buyRequestAmount, sellOrder.Utxo.Amount)
	buyShouldPayAmount := calcFromAmountByToAmount(buyReceiveAmount, buyContractArgs)
	
	sellContractArgs := DecodeDexProgram(sellOrder.Utxo.ControlProgram)
	sellRequestAmount := calcToAmountByFromAmount(sellOrder.Utxo.Amount, sellContractArgs)
	sellReceiveAmount := min(sellRequestAmount, buyOrder.Utxo.Amount)
	sellShouldPayAmount := calcFromAmountByToAmount(sellReceiveAmount, sellContractArgs)

	partialTradeStatus := make([]bool, 2)
	if buyOrder.ToAssetID.String() > buyOrder.FromAssetID.String() {
		partialTradeStatus[0] = addMatchTxOutput(&txData, buyOrder, buyReceiveAmount, buyShouldPayAmount, buyContractArgs.SellerProgram)
		partialTradeStatus[1] = addMatchTxOutput(&txData, sellOrder, sellReceiveAmount, sellShouldPayAmount, sellContractArgs.SellerProgram)
	} else {
		partialTradeStatus[1] = addMatchTxOutput(&txData, sellOrder, sellReceiveAmount, sellShouldPayAmount, sellContractArgs.SellerProgram)
		partialTradeStatus[0] = addMatchTxOutput(&txData, buyOrder, buyReceiveAmount, buyShouldPayAmount, buyContractArgs.SellerProgram)
	}

	addMatchTxFeeOutput(&txData, buyShouldPayAmount, sellReceiveAmount, *buyOrder.ToAssetID)
	addMatchTxFeeOutput(&txData, sellShouldPayAmount, buyReceiveAmount, *sellOrder.ToAssetID)

	tx := types.NewTx(txData)
	setMatchTxArguments(tx, buyReceiveAmount, sellReceiveAmount, partialTradeStatus)
	return tx, nil
}

// addMatchTxOutput return whether partial matched
func addMatchTxOutput(txData *types.TxData, order *common.Order, receiveAmount, shouldPayAmount uint64, receiveProgram []byte) bool {
	txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*order.ToAssetID, receiveAmount, receiveProgram))
	if order.Utxo.Amount > shouldPayAmount {
		txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*order.FromAssetID, order.Utxo.Amount-shouldPayAmount, order.Utxo.ControlProgram))
		return true
	}
	return false
}

func addMatchTxFeeOutput(txData *types.TxData, shouldPayAmount, oppositeReceiveAmount uint64, toAssetID bc.AssetID) {
	if shouldPayAmount > oppositeReceiveAmount {
		txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(toAssetID, shouldPayAmount-oppositeReceiveAmount, []byte{ /** node address */ }))
	}
}

func setMatchTxArguments(tx *types.Tx, buyReceiveAmount, sellReceiveAmount uint64, partialTradeStatus []bool) {
	clauseSelectors := make([][]byte, 2)
	for i, isPartial := range partialTradeStatus {
		if !isPartial {
			clauseSelectors[i] = vm.Int64Bytes(1)
		}
	}
	tx.SetInputArguments(0, [][]byte{vm.Int64Bytes(int64(buyReceiveAmount)), clauseSelectors[0]})
	tx.SetInputArguments(1, [][]byte{vm.Int64Bytes(int64(sellReceiveAmount)), clauseSelectors[1]})
}

func adjustOrderTable(tx *types.Tx, buyOrders, sellOrders *vprCommon.Stack) error {
	buyOrder := buyOrders.Pop().(*common.Order)
	sellOrder := sellOrders.Pop().(*common.Order)

	if hex.EncodeToString(tx.Outputs[buyOrderOutputIndex].ControlProgram()) == hex.EncodeToString(tx.Inputs[0].ControlProgram()) {
		order, err := common.OutputToOrder(tx, buyOrderOutputIndex)
		if err != nil {
			return err
		}

		buyOrders.Push(order)
		return nil
	}

	if hex.EncodeToString(tx.Outputs[sellOrderOutputIndex].ControlProgram()) == hex.EncodeToString(tx.Inputs[1].ControlProgram()) {
		order, err := common.OutputToOrder(tx, sellOrderOutputIndex)
		if err != nil {
			return err
		}

		sellOrders.Push(order)
	}
	return nil
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
