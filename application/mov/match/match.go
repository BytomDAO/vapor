package match

import (
	"encoding/hex"
	"math"
	"math/big"

	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/application/mov/contract"
	"github.com/bytom/vapor/consensus/segwit"
	"github.com/bytom/vapor/errors"
	vprMath "github.com/bytom/vapor/math"
	"github.com/bytom/vapor/math/decimal"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/vm"
	"github.com/bytom/vapor/protocol/vm/vmutil"
)

// Engine is used to generate math transactions
type Engine struct {
	orderBook   *OrderBook
	maxFeeRate  float64
	nodeProgram []byte
}

// NewEngine return a new Engine
func NewEngine(orderBook *OrderBook, maxFeeRate float64, nodeProgram []byte) *Engine {
	return &Engine{orderBook: orderBook, maxFeeRate: maxFeeRate, nodeProgram: nodeProgram}
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

func (e *Engine) addMatchTxFeeOutput(txData *types.TxData) error {
	txFee, err := CalcMatchedTxFee(txData, e.maxFeeRate)
	if err != nil {
		return err
	}

	for assetID, matchTxFee := range txFee {
		feeAmount, reminder := matchTxFee.FeeAmount, int64(0)
		if matchTxFee.FeeAmount > matchTxFee.MaxFeeAmount {
			feeAmount = matchTxFee.MaxFeeAmount
			reminder = matchTxFee.FeeAmount - matchTxFee.MaxFeeAmount
		}
		txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(assetID, uint64(feeAmount), e.nodeProgram))

		// There is the remaining amount after paying the handling fee, assign it evenly to participants in the transaction
		averageAmount := reminder / int64(len(txData.Inputs))
		if averageAmount == 0 {
			averageAmount = 1
		}

		for i := 0; i < len(txData.Inputs) && reminder > 0; i++ {
			contractArgs, err := segwit.DecodeP2WMCProgram(txData.Inputs[i].ControlProgram())
			if err != nil {
				return err
			}

			if i == len(txData.Inputs)-1 {
				txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(assetID, uint64(reminder), contractArgs.SellerProgram))
			} else {
				txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(assetID, uint64(averageAmount), contractArgs.SellerProgram))
			}
			reminder -= averageAmount
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

		if err := e.orderBook.AddOrder(order); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) buildMatchTx(orders []*common.Order) (*types.Tx, error) {
	txData := &types.TxData{Version: 1}
	for i, order := range orders {
		input := types.NewSpendInput(nil, *order.Utxo.SourceID, *order.FromAssetID, order.Utxo.Amount, order.Utxo.SourcePos, order.Utxo.ControlProgram)
		txData.Inputs = append(txData.Inputs, input)

		oppositeOrder := orders[calcOppositeIndex(len(orders), i)]
		if err := addMatchTxOutput(txData, input, order, oppositeOrder.Utxo.Amount); err != nil {
			return nil, err
		}
	}

	if err := e.addMatchTxFeeOutput(txData); err != nil {
		return nil, err
	}

	byteData, err := txData.MarshalText()
	if err != nil {
		return nil, err
	}

	txData.SerializedSize = uint64(len(byteData))
	return types.NewTx(*txData), nil
}

// MatchedTxFee is object to record the mov tx's fee information
type MatchedTxFee struct {
	MaxFeeAmount int64
	FeeAmount    int64
}

// CalcMatchedTxFee is used to calculate tx's MatchedTxFees
func CalcMatchedTxFee(txData *types.TxData, maxFeeRate float64) (map[bc.AssetID]*MatchedTxFee, error) {
	assetFeeMap := make(map[bc.AssetID]*MatchedTxFee)
	dealProgMaps := make(map[string]bool)

	for _, input := range txData.Inputs {
		assetFeeMap[input.AssetID()] = &MatchedTxFee{FeeAmount: int64(input.AssetAmount().Amount)}
		contractArgs, err := segwit.DecodeP2WMCProgram(input.ControlProgram())
		if err != nil {
			return nil, err
		}

		dealProgMaps[hex.EncodeToString(contractArgs.SellerProgram)] = true
	}

	for _, input := range txData.Inputs {
		contractArgs, err := segwit.DecodeP2WMCProgram(input.ControlProgram())
		if err != nil {
			return nil, err
		}

		oppositeAmount := uint64(assetFeeMap[contractArgs.RequestedAsset].FeeAmount)
		receiveAmount := vprMath.MinUint64(CalcRequestAmount(input.Amount(), contractArgs), oppositeAmount)
		assetFeeMap[input.AssetID()].MaxFeeAmount = calcMaxFeeAmount(calcShouldPayAmount(receiveAmount, contractArgs), maxFeeRate)
	}

	for _, output := range txData.Outputs {
		assetAmount := output.AssetAmount()
		if _, ok := dealProgMaps[hex.EncodeToString(output.ControlProgram())]; ok || segwit.IsP2WMCScript(output.ControlProgram()) {
			assetFeeMap[*assetAmount.AssetId].FeeAmount -= int64(assetAmount.Amount)
			if assetFeeMap[*assetAmount.AssetId].FeeAmount <= 0 {
				delete(assetFeeMap, *assetAmount.AssetId)
			}
		}
	}
	return assetFeeMap, nil
}

func addMatchTxOutput(txData *types.TxData, txInput *types.TxInput, order *common.Order, oppositeAmount uint64) error {
	contractArgs, err := segwit.DecodeP2WMCProgram(order.Utxo.ControlProgram)
	if err != nil {
		return err
	}

	requestAmount := CalcRequestAmount(order.Utxo.Amount, contractArgs)
	receiveAmount := vprMath.MinUint64(requestAmount, oppositeAmount)
	shouldPayAmount := calcShouldPayAmount(receiveAmount, contractArgs)
	isPartialTrade := requestAmount > receiveAmount

	setMatchTxArguments(txInput, isPartialTrade, len(txData.Outputs), receiveAmount)
	txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*order.ToAssetID, receiveAmount, contractArgs.SellerProgram))
	if isPartialTrade {
		txData.Outputs = append(txData.Outputs, types.NewIntraChainOutput(*order.FromAssetID, order.Utxo.Amount-shouldPayAmount, order.Utxo.ControlProgram))
	}
	return nil
}

func CalcRequestAmount(fromAmount uint64, contractArg *vmutil.MagneticContractArgs) uint64 {
	res := big.NewInt(0).SetUint64(fromAmount)
	res.Mul(res, big.NewInt(contractArg.RatioNumerator)).Quo(res, big.NewInt(contractArg.RatioDenominator))
	if !res.IsUint64() {
		return 0
	}
	return res.Uint64()
}

func calcShouldPayAmount(receiveAmount uint64, contractArg *vmutil.MagneticContractArgs) uint64 {
	res := big.NewInt(0).SetUint64(receiveAmount)
	res.Mul(res, big.NewInt(contractArg.RatioDenominator)).Quo(res, big.NewInt(contractArg.RatioNumerator))
	if !res.IsUint64() {
		return 0
	}
	return res.Uint64()
}

func calcMaxFeeAmount(shouldPayAmount uint64, maxFeeRate float64) int64 {
	return int64(math.Ceil(float64(shouldPayAmount) * maxFeeRate))
}

func calcOppositeIndex(size int, selfIdx int) int {
	return (selfIdx + 1) % size
}

func IsMatched(orders []*common.Order) bool {
	sortedOrders := sortOrders(orders)
	if len(sortedOrders) == 0 {
		return false
	}

	rate := decimal.New(1, 0).Div(orderRatio(sortedOrders[0]))
	oppositeRate := decimal.New(1, 0)
	for i := 1; i < len(sortedOrders); i++ {
		oppositeRate = oppositeRate.Mul(orderRatio(sortedOrders[i]))
	}
	return rate.Cmp(oppositeRate) >= 0
}

func orderRatio(order *common.Order) *decimal.Decimal {
	numerator := decimal.New(order.RatioNumerator, 0)
	denominator := decimal.New(order.RatioDenominator, 0)
	return numerator.Div(denominator)
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
