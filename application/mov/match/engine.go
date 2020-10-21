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

type orderPosition struct {
	blockHeight uint64
	txIndex     uint64
}

// NewEngine return a new Engine
func NewEngine(orderBook *OrderBook, rewardProgram []byte) *Engine {
	return &Engine{orderBook: orderBook, feeStrategy: NewDefaultFeeStrategy(), rewardProgram: rewardProgram}
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

	tx, partialOrderPositions, err := e.buildMatchTx(sortOrders(e.orderBook.PeekOrders(tradePairs)))
	if err != nil {
		return nil, err
	}

	for _, tradePair := range tradePairs {
		e.orderBook.PopOrder(tradePair)
	}

	if err := e.addReOrder(tx, partialOrderPositions); err != nil {
		return nil, err
	}
	return tx, nil
}

func addMatchTxFeeOutput(txData *types.TxData, fees []*bc.AssetAmount, rewardProgram []byte) {
	for _, feeAmount := range fees {
		txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*feeAmount.AssetId, feeAmount.Amount, rewardProgram))
	}
}

func (e *Engine) addReOrder(tx *types.Tx, partialOrderPositions []*orderPosition) error {
	index := 0
	for i, output := range tx.Outputs {
		if !segwit.IsP2WMCScript(output.ControlProgram()) || output.AssetAmount().Amount == 0 {
			continue
		}

		partialOrderPos := partialOrderPositions[index]
		order, err := common.NewOrderFromOutput(tx, i, partialOrderPos.blockHeight, partialOrderPos.txIndex)
		if err != nil {
			return err
		}

		index++
		e.orderBook.AddOrder(order)
	}
	return nil
}

func addRefundOutput(txData *types.TxData, takerProgram []byte) {
	refundAmount := map[bc.AssetID]uint64{}
	var assetIDs []bc.AssetID
	for _, input := range txData.Inputs {
		refundAmount[input.AssetID()] += input.Amount()
		assetIDs = append(assetIDs, input.AssetID())
	}

	for _, output := range txData.Outputs {
		assetAmount := output.AssetAmount()
		refundAmount[*assetAmount.AssetId] -= assetAmount.Amount
	}

	for assetID, amount := range refundAmount {
		if amount == 0 {
			continue
		}

		txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(assetID, amount, takerProgram))
	}
}

func (e *Engine) buildMatchTx(orders []*common.Order) (*types.Tx, []*orderPosition, error) {
	txData := &types.TxData{Version: 1}
	for _, order := range orders {
		input := types.NewSpendInput(nil, *order.Utxo.SourceID, *order.FromAssetID, order.Utxo.Amount, order.Utxo.SourcePos, order.Utxo.ControlProgram)
		txData.Inputs = append(txData.Inputs, input)
	}

	takerPos := takerPos(orders)
	receivedAmounts, priceDiffs := CalcReceivedAmount(orders)
	allocatedAssets := e.feeStrategy.Allocate(receivedAmounts, priceDiffs, takerPos)

	partialOrderPositions, err := addMatchTxOutput(txData, orders, receivedAmounts, allocatedAssets)
	if err != nil {
		return nil, nil, err
	}

	addMatchTxFeeOutput(txData, allocatedAssets.Fees, e.rewardProgram)
	addRefundOutput(txData, orders[takerPos].SellerProgram)

	byteData, err := txData.MarshalText()
	if err != nil {
		return nil, nil, err
	}

	txData.SerializedSize = uint64(len(byteData))
	return types.NewTx(*txData), partialOrderPositions, nil
}

func addMatchTxOutput(txData *types.TxData, orders []*common.Order, receivedAmounts []*bc.AssetAmount, allocatedAssets *AllocatedAssets) ([]*orderPosition, error) {
	var partialOrderPositions []*orderPosition
	for i, order := range orders {
		receivedAmount := receivedAmounts[i].Amount
		shouldPayAmount := calcShouldPayAmount(receivedAmount, order.RatioNumerator, order.RatioDenominator)

		requestAmount := CalcRequestAmount(order.Utxo.Amount, order.RatioNumerator, order.RatioDenominator)
		exchangeAmount := order.Utxo.Amount - shouldPayAmount
		isPartialTrade := requestAmount > receivedAmount && CalcRequestAmount(exchangeAmount, order.RatioNumerator, order.RatioDenominator) >= 1

		setMatchTxArguments(txData.Inputs[i], isPartialTrade, len(txData.Outputs), receivedAmount)

		txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*order.ToAssetID, allocatedAssets.Receives[i].Amount, order.SellerProgram))
		if isPartialTrade {
			txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*order.FromAssetID, exchangeAmount, order.Utxo.ControlProgram))
			partialOrderPositions = append(partialOrderPositions, &orderPosition{blockHeight: order.BlockHeight, txIndex: order.TxIndex})
		}
	}
	return partialOrderPositions, nil
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
func CalcReceivedAmount(orders []*common.Order) ([]*bc.AssetAmount, []*bc.AssetAmount) {
	var receivedAmounts, priceDiffs, shouldPayAmounts []*bc.AssetAmount
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
		priceDiffs = append(priceDiffs, &bc.AssetAmount{AssetId: oppositeShouldPayAmount.AssetId, Amount: 0})
		if oppositeShouldPayAmount.Amount > receivedAmount.Amount {
			priceDiffs[i].Amount = oppositeShouldPayAmount.Amount - receivedAmount.Amount
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

	product := big.NewRat(1, 1)
	for _, order := range orders {
		product.Mul(product, big.NewRat(order.RatioNumerator, order.RatioDenominator))
	}
	one := big.NewRat(1, 1)
	return product.Cmp(one) <= 0
}

func takerPos(orders []*common.Order) int {
	for i, order := range orders {
		if !isMaker(order, orders[calcOppositeIndex(len(orders), i)]) {
			return i
		}
	}
	return 0
}

func isMaker(order, oppositeOrder *common.Order) bool {
	if order.BlockHeight != oppositeOrder.BlockHeight {
		return order.BlockHeight < oppositeOrder.BlockHeight
	}

	if order.TxIndex != oppositeOrder.TxIndex {
		return order.TxIndex < oppositeOrder.TxIndex
	}

	return order.UTXOHash().String() < oppositeOrder.UTXOHash().String()
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
