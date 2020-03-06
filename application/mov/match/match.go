package match

import (
	"math/big"

	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/application/mov/contract"
	"github.com/bytom/vapor/consensus/segwit"
	"github.com/bytom/vapor/errors"
	vprMath "github.com/bytom/vapor/math"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/vm"
)

// Engine is used to generate math transactions
type Engine struct {
	orderBook     *OrderBook
	feeStrategy   FeeStrategy
	rewardProgram []byte
}

// NewEngine return a new Engine
func NewEngine(orderBook *OrderBook, feeStrategy FeeStrategy, rewardProgram []byte) *Engine {
	return &Engine{orderBook: orderBook, feeStrategy: feeStrategy, rewardProgram: rewardProgram}
}

// HasMatchedTx check does the input trade pair can generate a match deal
func (e *Engine) HasMatchedTx(tradePairs ...*common.TradePair) bool {
	if err := validateTradePairs(tradePairs); err != nil {
		return false
	}

	orders := e.orderBook.PeekOrders(tradePairs)
	if len(orders) == 0 {
		return false
	}

	return IsMatched(orders)
}

// NextMatchedTx return the next matchable transaction by the specified trade pairs
// the size of trade pairs at least 2, and the sequence of trade pairs can form a loop
// for example, [assetA -> assetB, assetB -> assetC, assetC -> assetA]
func (e *Engine) NextMatchedTx(tradePairs ...*common.TradePair) (*types.Tx, error) {
	if !e.HasMatchedTx(tradePairs...) {
		return nil, errors.New("the specified trade pairs can not be matched")
	}

	tx, err := e.buildMatchTx(sortOrders(e.orderBook.PeekOrders(tradePairs)))
	if err != nil {
		return nil, err
	}

	for _, tradePair := range tradePairs {
		e.orderBook.PopOrder(tradePair)
	}

	if err := e.addPartialTradeOrder(tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (e *Engine) addMatchTxFeeOutput(txData *types.TxData, refunds []RefundAssets, fees []*bc.AssetAmount) error {
	for _, feeAmount := range fees {
		txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*feeAmount.AssetId, feeAmount.Amount, e.rewardProgram))
	}

	for i, refund := range refunds {
		for _, assetAmount := range refund {
			contractArgs, err := segwit.DecodeP2WMCProgram(txData.Inputs[i].ControlProgram())
			if err != nil {
				return err
			}

			txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*assetAmount.AssetId, assetAmount.Amount, contractArgs.SellerProgram))
		}
	}
	return nil
}

func (e *Engine) addPartialTradeOrder(tx *types.Tx) error {
	for i, output := range tx.Outputs {
		if !segwit.IsP2WMCScript(output.ControlProgram()) {
			continue
		}

		order, err := common.NewOrderFromOutput(tx, i)
		if err != nil {
			return err
		}

		e.orderBook.AddOrder(order)
	}
	return nil
}

func (e *Engine) buildMatchTx(orders []*common.Order) (*types.Tx, error) {
	txData := &types.TxData{Version: 1}
	for _, order := range orders {
		input := types.NewSpendInput(nil, *order.Utxo.SourceID, *order.FromAssetID, order.Utxo.Amount, order.Utxo.SourcePos, order.Utxo.ControlProgram)
		txData.Inputs = append(txData.Inputs, input)
	}

	receivedAmounts, priceDiffs := CalcReceivedAmount(orders)
	allocatedAssets := e.feeStrategy.Allocate(receivedAmounts, priceDiffs)
	if err := addMatchTxOutput(txData, orders, receivedAmounts, allocatedAssets.Received); err != nil {
		return nil, err
	}

	if err := e.addMatchTxFeeOutput(txData, allocatedAssets.Refunds, allocatedAssets.Fees); err != nil {
		return nil, err
	}

	byteData, err := txData.MarshalText()
	if err != nil {
		return nil, err
	}

	txData.SerializedSize = uint64(len(byteData))
	return types.NewTx(*txData), nil
}

func addMatchTxOutput(txData *types.TxData, orders []*common.Order, receivedAmounts, receivedAfterDeductFee []*bc.AssetAmount) error {
	for i, order := range orders {
		contractArgs, err := segwit.DecodeP2WMCProgram(order.Utxo.ControlProgram)
		if err != nil {
			return err
		}

		requestAmount := CalcRequestAmount(order.Utxo.Amount, contractArgs.RatioNumerator, contractArgs.RatioDenominator)
		receivedAmount := receivedAmounts[i].Amount
		shouldPayAmount := calcShouldPayAmount(receivedAmount, contractArgs.RatioNumerator, contractArgs.RatioDenominator)
		isPartialTrade := requestAmount > receivedAmount

		setMatchTxArguments(txData.Inputs[i], isPartialTrade, len(txData.Outputs), receivedAfterDeductFee[i].Amount)
		txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*order.ToAssetID, receivedAfterDeductFee[i].Amount, contractArgs.SellerProgram))
		if isPartialTrade {
			txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*order.FromAssetID, order.Utxo.Amount-shouldPayAmount, order.Utxo.ControlProgram))
		}
	}
	return nil
}

func calcOppositeIndex(size int, selfIdx int) int {
	return (selfIdx + 1) % size
}

// CalcRequestAmount is from amount * numerator / ratioDenominator
func CalcRequestAmount(fromAmount uint64, ratioNumerator, ratioDenominator int64) uint64 {
	res := big.NewInt(0).SetUint64(fromAmount)
	res.Mul(res, big.NewInt(ratioNumerator)).Quo(res, big.NewInt(ratioDenominator))
	if !res.IsUint64() {
		return 0
	}
	return res.Uint64()
}

func calcShouldPayAmount(receiveAmount uint64, ratioNumerator, ratioDenominator int64) uint64 {
	res := big.NewInt(0).SetUint64(receiveAmount)
	res.Mul(res, big.NewInt(ratioDenominator)).Quo(res, big.NewInt(ratioNumerator))
	if !res.IsUint64() {
		return 0
	}
	return res.Uint64()
}

// CalcReceivedAmount return amount of assets received by each participant in the matching transaction and the price difference
func CalcReceivedAmount(orders []*common.Order) ([]*bc.AssetAmount, map[bc.AssetID]int64) {
	priceDiffs := make(map[bc.AssetID]int64)
	var receivedAmounts, shouldPayAmounts []*bc.AssetAmount
	for i, order := range orders {
		requestAmount := CalcRequestAmount(order.Utxo.Amount, order.RatioNumerator, order.RatioDenominator)
		oppositeOrder := orders[calcOppositeIndex(len(orders), i)]
		receiveAmount := vprMath.MinUint64(oppositeOrder.Utxo.Amount, requestAmount)
		shouldPayAmount := calcShouldPayAmount(receiveAmount, order.RatioNumerator, order.RatioDenominator)
		receivedAmounts = append(receivedAmounts, &bc.AssetAmount{AssetId: order.ToAssetID, Amount: receiveAmount})
		shouldPayAmounts = append(shouldPayAmounts, &bc.AssetAmount{AssetId: order.FromAssetID, Amount: shouldPayAmount})
	}

	for i, receivedAmount := range receivedAmounts {
		oppositeShouldPayAmount := shouldPayAmounts[calcOppositeIndex(len(orders), i)]
		if oppositeShouldPayAmount.Amount > receivedAmount.Amount {
			assetId := oppositeShouldPayAmount.AssetId
			amount := oppositeShouldPayAmount.Amount - receivedAmount.Amount
			priceDiffs[*assetId] = int64(amount)
		}
	}
	return receivedAmounts, priceDiffs
}

// IsMatched check does the orders can be exchange
func IsMatched(orders []*common.Order) bool {
	sortedOrders := sortOrders(orders)
	if len(sortedOrders) == 0 {
		return false
	}

	rate := big.NewRat(sortedOrders[0].RatioDenominator, sortedOrders[0].RatioNumerator)
	oppositeRate := big.NewRat(1, 1)
	for i := 1; i < len(sortedOrders); i++ {
		oppositeRate.Mul(oppositeRate, big.NewRat(sortedOrders[i].RatioNumerator, sortedOrders[i].RatioDenominator))
	}

	return rate.Cmp(oppositeRate) >= 0
}

func setMatchTxArguments(txInput *types.TxInput, isPartialTrade bool, position int, receiveAmounts uint64) {
	var arguments [][]byte
	if isPartialTrade {
		arguments = [][]byte{vm.Int64Bytes(int64(receiveAmounts)), vm.Int64Bytes(int64(position)), vm.Int64Bytes(contract.PartialTradeClauseSelector)}
	} else {
		arguments = [][]byte{vm.Int64Bytes(int64(position)), vm.Int64Bytes(contract.FullTradeClauseSelector)}
	}
	txInput.SetArguments(arguments)
}

func validateTradePairs(tradePairs []*common.TradePair) error {
	if len(tradePairs) < 2 {
		return errors.New("size of trade pairs at least 2")
	}

	assetMap := make(map[string]bool)
	for _, tradePair := range tradePairs {
		assetMap[tradePair.FromAssetID.String()] = true
		if *tradePair.FromAssetID == *tradePair.ToAssetID {
			return errors.New("from asset id can't equal to asset id")
		}
	}

	for _, tradePair := range tradePairs {
		key := tradePair.ToAssetID.String()
		if _, ok := assetMap[key]; !ok {
			return errors.New("invalid trade pairs")
		}
		delete(assetMap, key)
	}
	return nil
}

func sortOrders(orders []*common.Order) []*common.Order {
	if len(orders) == 0 {
		return nil
	}

	orderMap := make(map[bc.AssetID]*common.Order)
	firstOrder := orders[0]
	for i := 1; i < len(orders); i++ {
		orderMap[*orders[i].FromAssetID] = orders[i]
	}

	sortedOrders := []*common.Order{firstOrder}
	for order := firstOrder; *order.ToAssetID != *firstOrder.FromAssetID; {
		nextOrder, ok := orderMap[*order.ToAssetID]
		if !ok {
			return nil
		}

		sortedOrders = append(sortedOrders, nextOrder)
		order = nextOrder
	}
	return sortedOrders
}
