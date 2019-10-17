package match

import (
	"math"
	"math/big"

	"github.com/vapor/application/mov/database"
	"github.com/vapor/application/mov/util"
	"github.com/vapor/application/mov/common"
	"github.com/vapor/consensus/segwit"
	vprMath "github.com/vapor/math"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm"
	"github.com/vapor/protocol/vm/vmutil"
)

const (
	partialBuyOrderOutputIdx  = 1
	partialSellOrderOutputIdx = 2
)

type Engine struct {
	orderTable *OrderTable
}

func NewEngine(movStore database.MovStore) *Engine {
	return &Engine{orderTable: NewOrderTable(movStore)}
}

// NextMatchedTx match two opposite pending orders.
// for example, the buy orders want change A with B, then the sell orders must change B with A.
func (e *Engine) NextMatchedTx(buyTradePair, sellTradePair *common.TradePair) (*types.Tx, error) {
	buyOrder, err := e.orderTable.PeekOrder(buyTradePair)
	if err != nil {
		return nil, err
	}

	sellOrder, err := e.orderTable.PeekOrder(sellTradePair)
	if err != nil {
		return nil, err
	}

	if buyOrder == nil || sellOrder == nil {
		return nil, nil
	}

	buyContractArgs, err := segwit.DecodeP2WMCProgram(buyOrder.Utxo.ControlProgram)
	if err != nil {
		return nil, err
	}

	sellContractArgs, err := segwit.DecodeP2WMCProgram(sellOrder.Utxo.ControlProgram)
	if err != nil {
		return nil, err
	}

	if canNotBeMatched(buyOrder, sellOrder, buyContractArgs, sellContractArgs) {
		return nil, nil
	}

	tx := buildMatchTx(buyOrder, sellOrder, buyContractArgs, sellContractArgs)

	e.orderTable.PopOrder(buyTradePair)
	e.orderTable.PopOrder(sellTradePair)
	if err := addPartialTradeOrder(tx, e.orderTable, buyTradePair, sellTradePair); err != nil {
		return nil, err
	}
	return tx, nil
}

func canNotBeMatched(buyOrder, sellOrder *common.Order, buyContractArgs, sellContractArgs *vmutil.MagneticContractArgs) bool {
	if buyOrder.ToAssetID != sellOrder.FromAssetID || sellOrder.ToAssetID != buyOrder.FromAssetID {
		return false
	}

	if buyContractArgs.RatioNumerator == 0 || sellContractArgs.RatioDenominator == 0 {
		return false
	}

	buyRate := big.NewFloat(0).Quo(big.NewFloat(0).SetInt64(buyContractArgs.RatioDenominator), big.NewFloat(0).SetInt64(buyContractArgs.RatioNumerator))
	sellRate := big.NewFloat(0).Quo(big.NewFloat(0).SetInt64(sellContractArgs.RatioNumerator), big.NewFloat(0).SetInt64(sellContractArgs.RatioDenominator))
	return buyRate.Cmp(sellRate) < 0
}

func buildMatchTx(buyOrder, sellOrder *common.Order, buyContractArgs, sellContractArgs *vmutil.MagneticContractArgs) *types.Tx {
	txData := &types.TxData{}
	txData.Inputs = append(txData.Inputs, types.NewSpendInput(nil, *buyOrder.Utxo.SourceID, *buyOrder.FromAssetID, buyOrder.Utxo.Amount, buyOrder.Utxo.SourcePos, buyOrder.Utxo.ControlProgram))
	txData.Inputs = append(txData.Inputs, types.NewSpendInput(nil, *sellOrder.Utxo.SourceID, *sellOrder.FromAssetID, sellOrder.Utxo.Amount, sellOrder.Utxo.SourcePos, sellOrder.Utxo.ControlProgram))

	buyRequestAmount := calcRequestAmount(buyOrder.Utxo.Amount, buyContractArgs)
	buyReceiveAmount := vprMath.MinUint64(buyRequestAmount, sellOrder.Utxo.Amount)
	buyShouldPayAmount := calcShouldPayAmount(buyReceiveAmount, buyContractArgs)

	sellRequestAmount := calcRequestAmount(sellOrder.Utxo.Amount, sellContractArgs)
	sellReceiveAmount := vprMath.MinUint64(sellRequestAmount, buyOrder.Utxo.Amount)
	sellShouldPayAmount := calcShouldPayAmount(sellReceiveAmount, sellContractArgs)

	addMatchTxOutput(txData, buyOrder, buyReceiveAmount, buyShouldPayAmount, buyContractArgs.SellerProgram)
	addMatchTxOutput(txData, sellOrder, sellReceiveAmount, sellShouldPayAmount, sellContractArgs.SellerProgram)

	addMatchTxFeeOutput(txData, buyShouldPayAmount, sellReceiveAmount, *buyOrder.FromAssetID)
	addMatchTxFeeOutput(txData, sellShouldPayAmount, buyReceiveAmount, *sellOrder.FromAssetID)

	setMatchTxArguments(txData, buyReceiveAmount, sellReceiveAmount)
	tx := types.NewTx(*txData)
	return tx
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

func addMatchTxFeeOutput(txData *types.TxData, shouldPayAmount, oppositeReceiveAmount uint64, fromAssetID bc.AssetID) {
	if shouldPayAmount > oppositeReceiveAmount {
		txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(fromAssetID, shouldPayAmount-oppositeReceiveAmount, []byte{ /** node address */ }))
	}
}

func setMatchTxArguments(txData *types.TxData, buyReceiveAmount, sellReceiveAmount uint64) {
	partialTradeStatus := make([]bool, 2)
	partialTradeStatus[0] = segwit.IsP2WMCScript(txData.Outputs[partialBuyOrderOutputIdx].ControlProgram())
	partialTradeStatus[1] = len(txData.Outputs) > 2 && segwit.IsP2WMCScript(txData.Outputs[partialSellOrderOutputIdx].ControlProgram())

	receiveAmounts := []uint64{buyReceiveAmount, sellReceiveAmount}
	arguments := make([][][]byte, 2)
	var position int64
	for i, isPartial := range partialTradeStatus {
		if isPartial {
			arguments[i] = [][]byte{vm.Int64Bytes(int64(receiveAmounts[i])), vm.Int64Bytes(position), vm.Int64Bytes(0)}
			position += 2
		} else {
			arguments[i] = [][]byte{vm.Int64Bytes(position), vm.Int64Bytes(1)}
			position++
		}
		txData.Inputs[i].SetArguments(arguments[i])
	}
}

func addPartialTradeOrder(tx *types.Tx, orderTable *OrderTable, buyTradePair, sellTradePair *common.TradePair) error {
	if util.IsPartialTradeClauseSelector(tx.Inputs[0]) {
		order, err := common.NewOrderFromOutput(tx, partialBuyOrderOutputIdx)
		if err != nil {
			return err
		}

		if err := orderTable.AddOrder(buyTradePair, order); err != nil {
			return err
		}
	}
	if util.IsPartialTradeClauseSelector(tx.Inputs[1]) {
		order, err := common.NewOrderFromOutput(tx, partialSellOrderOutputIdx)
		if err != nil {
			return err
		}

		if err := orderTable.AddOrder(sellTradePair, order); err != nil {
			return err
		}
	}
	return nil
}

func calcRequestAmount(fromAmount uint64, contractArg *vmutil.MagneticContractArgs) uint64 {
	return uint64(int64(fromAmount) * contractArg.RatioNumerator / contractArg.RatioDenominator)
}

func calcShouldPayAmount(receiveAmount uint64, contractArg *vmutil.MagneticContractArgs) uint64 {
	return uint64(math.Ceil(float64(receiveAmount) * float64(contractArg.RatioDenominator) / float64(contractArg.RatioNumerator)))
}
