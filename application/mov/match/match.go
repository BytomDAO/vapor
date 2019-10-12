package match

import (
	"math/big"

	"github.com/vapor/application/mov/common"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm"
)

const (
	partialBuyOrderOutputIdx  = 1
	partialSellOrderOutputIdx = 2
)

// GenerateMatchedTxs match two opposite pending orders.
// for example, the buy orders want change A with B, then the sell orders must change B with A.
// the input order's rate must in descending order.
func GenerateMatchedTxs(orderTable *OrderTable) ([]*types.Tx, error) {
	var matchedTxs []*types.Tx
	for orderTable.HasNextOrder() {
		buyOrder, sellOrder := orderTable.PeekOrder()
		if canNotBeMatched(buyOrder, sellOrder) {
			break
		}

		tx, partialTradeStatus := buildMatchTx(buyOrder, sellOrder)
		matchedTxs = append(matchedTxs, tx)
		orderTable.PopOrder()
		if err := addPartialTradeOrder(tx, partialTradeStatus, orderTable); err != nil {
			return nil, err
		}
	}
	return matchedTxs, nil
}

func canNotBeMatched(buyOrder, sellOrder *common.Order) bool {
	if buyOrder.ToAssetID != sellOrder.FromAssetID || sellOrder.ToAssetID != buyOrder.FromAssetID {
		return false
	}

	buyContractArgs := DecodeDexProgram(buyOrder.Utxo.ControlProgram)
	sellContractArgs := DecodeDexProgram(sellOrder.Utxo.ControlProgram)

	if buyContractArgs.RatioMolecule == 0 || sellContractArgs.RatioDenominator == 0 {
		return false
	}

	buyRate := big.NewFloat(0).Quo(big.NewFloat(0).SetUint64(buyContractArgs.RatioDenominator), big.NewFloat(0).SetUint64(buyContractArgs.RatioMolecule))
	sellRate := big.NewFloat(0).Quo(big.NewFloat(0).SetUint64(sellContractArgs.RatioMolecule), big.NewFloat(0).SetUint64(buyContractArgs.RatioDenominator))
	return buyRate.Cmp(sellRate) < 0
}

func buildMatchTx(buyOrder, sellOrder *common.Order) (*types.Tx, []bool) {
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
	partialTradeStatus[0] = addMatchTxOutput(&txData, buyOrder, buyReceiveAmount, buyShouldPayAmount, buyContractArgs.SellerProgram)
	partialTradeStatus[1] = addMatchTxOutput(&txData, sellOrder, sellReceiveAmount, sellShouldPayAmount, sellContractArgs.SellerProgram)

	addMatchTxFeeOutput(&txData, buyShouldPayAmount, sellReceiveAmount, *buyOrder.ToAssetID)
	addMatchTxFeeOutput(&txData, sellShouldPayAmount, buyReceiveAmount, *sellOrder.ToAssetID)

	tx := types.NewTx(txData)
	setMatchTxArguments(tx, buyReceiveAmount, sellReceiveAmount, partialTradeStatus)
	return tx, partialTradeStatus
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
	receiveAmounts := []uint64{buyReceiveAmount, sellReceiveAmount}
	arguments := make([][][]byte, 2)
	var position int64 = 0

	for i, isPartial := range partialTradeStatus {
		if isPartial {
			arguments[i] = [][]byte{vm.Int64Bytes(int64(receiveAmounts[i])), vm.Int64Bytes(position), vm.Int64Bytes(0)}
			position += 2
		} else {
			arguments[i] = [][]byte{vm.Int64Bytes(position), vm.Int64Bytes(1)}
			position += 1
		}
		tx.SetInputArguments(uint32(i), arguments[i])
	}
}

func addPartialTradeOrder(tx *types.Tx, partialTradeStatus []bool, orderTable *OrderTable) error {
	if partialTradeStatus[0] {
		order, err := common.OutputToOrder(tx, partialBuyOrderOutputIdx)
		if err != nil {
			return err
		}

		orderTable.AddBuyOrder(order)
	}
	if partialTradeStatus[1] {
		order, err := common.OutputToOrder(tx, partialSellOrderOutputIdx)
		if err != nil {
			return err
		}

		orderTable.AddSellOrder(order)
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
