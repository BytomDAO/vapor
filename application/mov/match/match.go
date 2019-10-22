package match

import (
	"math"
	"math/big"

	"github.com/vapor/application/mov/common"
	"github.com/vapor/application/mov/database"
	"github.com/vapor/application/mov/util"
	"github.com/vapor/consensus/segwit"
	"github.com/vapor/errors"
	vprMath "github.com/vapor/math"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm"
	"github.com/vapor/protocol/vm/vmutil"
)

var maxFeeRate = 0.05

type Engine struct {
	orderTable  *OrderTable
	nodeProgram []byte
}

func NewEngine(movStore database.MovStore, nodeProgram []byte) *Engine {
	return &Engine{orderTable: NewOrderTable(movStore), nodeProgram: nodeProgram}
}

// NextMatchedTx return the next matchable transaction by the specified trade pairs
// the size of trade pairs at least, and the sequence of trade pairs can form a loop
// for example, [assetA -> assetB, assetB -> assetC, assetC -> assetA]
func (e *Engine) NextMatchedTx(tradePairs  ...*common.TradePair) (*types.Tx, error) {
	if err := validateTradePairs(tradePairs); err != nil {
		return nil, err
	}

	var orders []*common.Order
	for _, tradePair := range tradePairs {
		order := e.orderTable.PeekOrder(tradePair)
		if order == nil {
			return nil, nil
		}

		orders = append(orders, order)
	}

	tx, err := e.buildMatchTx(orders)
	if err != nil {
		return nil, err
	}

	if tx == nil {
		return nil, nil
	}

	for _, tradePair := range tradePairs {
		e.orderTable.PopOrder(tradePair)
	}
	if err := addPartialTradeOrder(tx, e.orderTable); err != nil {
		return nil, err
	}
	return tx, nil
}

func validateTradePairs(tradePairs []*common.TradePair) error {
	if len(tradePairs) < 2 {
		return errors.New("size of trade pairs at least 2")
	}

	for i, tradePair:= range tradePairs {
		oppositeTradePair := tradePairs[getOppositeIndex(len(tradePairs), i)]
		if *tradePair.FromAssetID != *oppositeTradePair.ToAssetID || *tradePair.ToAssetID != *oppositeTradePair.FromAssetID {
			return errors.New("specified trade pairs is invalid")
		}
	}
	return nil
}

func (e *Engine) buildMatchTx(orders []*common.Order) (*types.Tx, error) {
	txData := &types.TxData{Version: 1}
	var partialTradeStatus []bool
	var receiveAmounts []uint64

	for i, order := range orders {
		contractArgs, err := segwit.DecodeP2WMCProgram(order.Utxo.ControlProgram)
		if err != nil {
			return nil, err
		}

		oppositeOrder := orders[getOppositeIndex(len(orders), i)]
		oppositeContractArgs, err := segwit.DecodeP2WMCProgram(oppositeOrder.Utxo.ControlProgram)
		if err != nil {
			return nil, err
		}

		if canNotBeMatched(contractArgs, oppositeContractArgs) {
			return nil, nil
		}

		txData.Inputs = append(txData.Inputs, types.NewSpendInput(nil, *order.Utxo.SourceID, *order.FromAssetID, order.Utxo.Amount, order.Utxo.SourcePos, order.Utxo.ControlProgram))
		isPartialTrade, receiveAmount := addMatchTxOutput(txData, order, contractArgs, oppositeOrder.Utxo.Amount)
		partialTradeStatus = append(partialTradeStatus, isPartialTrade)
		receiveAmounts = append(receiveAmounts, receiveAmount)
	}

	setMatchTxArguments(txData, partialTradeStatus, receiveAmounts)
	if err := e.addMatchTxFeeOutput(txData); err != nil {
		return nil, err
	}

	byteData, err := txData.MarshalText()
	if err != nil {
		return nil, err
	}

	txData.SerializedSize = uint64(len(byteData))
	tx := types.NewTx(*txData)
	return tx, nil
}

func canNotBeMatched(contractArgs, oppositeContractArgs *vmutil.MagneticContractArgs) bool {
	if contractArgs.RatioNumerator == 0 || oppositeContractArgs.RatioDenominator == 0 {
		return false
	}

	buyRate := big.NewFloat(0).Quo(big.NewFloat(0).SetInt64(contractArgs.RatioDenominator), big.NewFloat(0).SetInt64(contractArgs.RatioNumerator))
	sellRate := big.NewFloat(0).Quo(big.NewFloat(0).SetInt64(oppositeContractArgs.RatioNumerator), big.NewFloat(0).SetInt64(oppositeContractArgs.RatioDenominator))
	return buyRate.Cmp(sellRate) < 0
}

// addMatchTxOutput return whether partial matched
func addMatchTxOutput(txData *types.TxData, order *common.Order, contractArgs *vmutil.MagneticContractArgs, oppositeAmount uint64) (bool, uint64) {
	requestAmount := calcRequestAmount(order.Utxo.Amount, contractArgs)
	receiveAmount := vprMath.MinUint64(requestAmount, oppositeAmount)
	shouldPayAmount := CalcShouldPayAmount(receiveAmount, contractArgs)

	txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*order.ToAssetID, receiveAmount, contractArgs.SellerProgram))
	if order.Utxo.Amount > shouldPayAmount {
		txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*order.FromAssetID, order.Utxo.Amount-shouldPayAmount, order.Utxo.ControlProgram))
		return true, receiveAmount
	}
	return false, receiveAmount
}

func (e *Engine) addMatchTxFeeOutput(txData *types.TxData) error {
	feeAssetAmountMap, err := CalcFeeFromMatchedTx(txData)
	if err != nil {
		return err
	}

	for feeAssetID, amount := range feeAssetAmountMap {
		var reminder uint64 = 0
		feeAmount := amount.payableFeeAmount
		if amount.payableFeeAmount > amount.maxFeeAmount {
			feeAmount = amount.maxFeeAmount
			reminder = amount.payableFeeAmount - amount.maxFeeAmount
		}
		txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(feeAssetID, feeAmount, e.nodeProgram))

		// There is the remaining amount after paying the handling fee, assign it evenly to participants in the transaction
		averageAmount := reminder / uint64(len(txData.Inputs))
		if averageAmount == 0 {
			averageAmount = 1
		}
		for i := 0; i < len(txData.Inputs) && reminder > 0; i++ {
			contractArgs, err := segwit.DecodeP2WMCProgram(txData.Inputs[i].ControlProgram())
			if err != nil {
				return err
			}

			if i == len(txData.Inputs)-1 {
				txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(feeAssetID, reminder, contractArgs.SellerProgram))
			} else {
				txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(feeAssetID, averageAmount, contractArgs.SellerProgram))
			}
			reminder -= averageAmount
		}
	}
	return nil
}

func setMatchTxArguments(txData *types.TxData, partialTradeStatus []bool, receiveAmounts []uint64) {
	var position int64
	for i, isPartial := range partialTradeStatus {
		var arguments [][]byte
		if isPartial {
			arguments = [][]byte{vm.Int64Bytes(int64(receiveAmounts[i])), vm.Int64Bytes(position), vm.Int64Bytes(0)}
			position += 2
		} else {
			arguments = [][]byte{vm.Int64Bytes(position), vm.Int64Bytes(1)}
			position++
		}
		txData.Inputs[i].SetArguments(arguments)
	}
}

func addPartialTradeOrder(tx *types.Tx, orderTable *OrderTable) error {
	for i, output := range tx.Outputs {
		if !segwit.IsP2WMCScript(output.ControlProgram()) {
			continue
		}

		order, err := common.NewOrderFromOutput(tx, i)
		if err != nil {
			return err
		}

		if err := orderTable.AddOrder(order); err != nil {
			return err
		}
	}
	return nil
}

func getOppositeIndex(size int, selfIdx int) int {
	oppositeIdx := selfIdx + 1
	if selfIdx >= size - 1 {
		oppositeIdx = 0
	}
	return oppositeIdx
}

func calcRequestAmount(fromAmount uint64, contractArg *vmutil.MagneticContractArgs) uint64 {
	return uint64(int64(fromAmount) * contractArg.RatioNumerator / contractArg.RatioDenominator)
}

func CalcShouldPayAmount(receiveAmount uint64, contractArg *vmutil.MagneticContractArgs) uint64 {
	return uint64(math.Ceil(float64(receiveAmount) * float64(contractArg.RatioDenominator) / float64(contractArg.RatioNumerator)))
}

func CalcMaxFeeAmount(shouldPayAmount uint64) uint64 {
	return uint64(math.Ceil(float64(shouldPayAmount) * maxFeeRate))
}

type feeAmount struct {
	maxFeeAmount     uint64
	payableFeeAmount uint64
}

func CalcFeeFromMatchedTx(txData *types.TxData) (map[bc.AssetID]*feeAmount, error) {
	assetAmountMap := make(map[bc.AssetID]*feeAmount)
	for _, input := range txData.Inputs {
		assetAmountMap[input.AssetID()] = &feeAmount{}
	}

	for _, input := range txData.Inputs {
		assetAmountMap[input.AssetID()].payableFeeAmount += input.AssetAmount().Amount
		outputPos, err := util.GetTradeReceivePosition(input)
		if err != nil {
			return nil, err
		}

		receiveOutput := txData.Outputs[outputPos]
		assetAmountMap[*receiveOutput.AssetAmount().AssetId].payableFeeAmount -= receiveOutput.AssetAmount().Amount
		contractArgs, err := segwit.DecodeP2WMCProgram(input.ControlProgram())
		if err != nil {
			return nil, err
		}

		assetAmountMap[input.AssetID()].maxFeeAmount = CalcMaxFeeAmount(CalcShouldPayAmount(receiveOutput.AssetAmount().Amount, contractArgs))
	}

	for _, output := range txData.Outputs {
		// minus the amount of the re-order
		if segwit.IsP2WMCScript(output.ControlProgram()) {
			assetAmountMap[*output.AssetAmount().AssetId].payableFeeAmount -= output.AssetAmount().Amount
		}
	}

	for assetID, amount := range assetAmountMap {
		if amount.payableFeeAmount == 0 {
			delete(assetAmountMap, assetID)
		}
	}
	return assetAmountMap, nil
}
