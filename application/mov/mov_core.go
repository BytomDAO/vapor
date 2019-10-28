package mov

import (
	"github.com/vapor/application/mov/common"
	"github.com/vapor/application/mov/contract"
	"github.com/vapor/application/mov/database"
	"github.com/vapor/application/mov/match"
	"github.com/vapor/consensus/segwit"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/errors"
	"github.com/vapor/math/checked"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const maxFeeRate = 0.05

var (
	errInvalidTradePairs = errors.New("The trade pairs in the tx input is invalid")
)

type MovCore struct {
	movStore database.MovStore
}

func NewMovCore(dbBackend, dbDir string) *MovCore {
	movDB := dbm.NewDB("mov", dbBackend, dbDir)
	return &MovCore{movStore: database.NewLevelDBMovStore(movDB)}
}

func (m *MovCore) Name() string {
	return "MOV"
}

func (m *MovCore) InitChainStatus(blockHeight uint64, blockHash *bc.Hash) error {
	return m.movStore.InitDBState(blockHeight, blockHash)
}

// ChainStatus return the current block height and block hash in dex core
func (m *MovCore) ChainStatus() (uint64, *bc.Hash, error) {
	state, err := m.movStore.GetMovDatabaseState()
	if err != nil {
		return 0, nil, err
	}

	return state.Height, state.Hash, nil
}

func (m *MovCore) ValidateBlock(block *types.Block, verifyResults []*bc.TxVerifyResult) error {
	return m.ValidateTxs(block.Transactions, verifyResults)
}

// ValidateTxs validate the trade transaction.
func (m *MovCore) ValidateTxs(txs []*types.Tx, verifyResults []*bc.TxVerifyResult) error {
	for _, tx := range txs {
		if common.IsMatchedTx(tx) {
			if err := validateMatchedTx(tx); err != nil {
				return err
			}
		}

		if common.IsCancelOrderTx(tx) {
			if err := validateCancelOrderTx(tx); err != nil {
				return err
			}
		}

		for _, output := range tx.Outputs {
			if !segwit.IsP2WMCScript(output.ControlProgram()) {
				continue
			}

			if err := validateMagneticContractArgs(output.AssetAmount().Amount, output.ControlProgram()); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateMatchedTx(tx *types.Tx) error {
	fromAssetIDMap := make(map[string]bool)
	toAssetIDMap := make(map[string]bool)
	for i, input := range tx.Inputs {
		if !segwit.IsP2WMCScript(input.ControlProgram()) {
			return errors.New("input program of matched tx must p2wmc script")
		}

		if contract.IsCancelClauseSelector(input) {
			return errors.New("can't exist cancel order in the matched transaction")
		}

		order, err := common.NewOrderFromInput(tx, i)
		if err != nil {
			return err
		}

		fromAssetIDMap[order.FromAssetID.String()] = true
		toAssetIDMap[order.ToAssetID.String()] = true
	}

	if len(fromAssetIDMap) != len(tx.Inputs) || len(toAssetIDMap) != len(tx.Inputs) {
		return errors.New("asset id must unique in matched transaction")
	}

	return validateMatchedTxFeeAmount(tx)
}

func validateCancelOrderTx(tx *types.Tx) error {
	for _, input := range tx.Inputs {
		if !segwit.IsP2WMCScript(input.ControlProgram()) {
			return errors.New("input program of cancel order tx must p2wmc script")
		}

		if contract.IsTradeClauseSelector(input) {
			return errors.New("can't exist trade order in the cancel order transaction")
		}
	}
	return nil
}

func validateMatchedTxFeeAmount(tx *types.Tx) error {
	txFee, err := match.CalcMatchedTxFee(&tx.TxData, maxFeeRate)
	if err != nil {
		return err
	}

	for _, amount := range txFee {
		if amount.FeeAmount > amount.MaxFeeAmount {
			return errors.New("amount of fee greater than max fee amount")
		}
	}
	return nil
}

func validateMagneticContractArgs(inputAmount uint64, program []byte) error {
	contractArgs, err := segwit.DecodeP2WMCProgram(program)
	if err != nil {
		return err
	}

	if contractArgs.RatioNumerator <= 0 || contractArgs.RatioDenominator <= 0 {
		return errors.New("ratio arguments must greater than zero")
	}

	if _, ok := checked.MulInt64(int64(inputAmount), contractArgs.RatioNumerator); !ok {
		return errors.New("ratio numerator of contract args product input amount is overflow")
	}
	return nil
}

// ApplyBlock parse pending order and cancel from the the transactions of block
// and add pending order to the dex db, remove cancel order from dex db.
func (m *MovCore) ApplyBlock(block *types.Block) error {
	if err := m.validateMatchedTxSequence(block.Transactions); err != nil {
		return err
	}

	addOrders, deleteOrders, err := applyTransactions(block.Transactions)
	if err != nil {
		return err
	}

	return m.movStore.ProcessOrders(addOrders, deleteOrders, &block.BlockHeader)
}

func (m *MovCore) validateMatchedTxSequence(txs []*types.Tx, ) error {
	matchEngine := match.NewEngine(m.movStore, maxFeeRate, nil)
	for _, matchedTx := range txs {
		if !common.IsMatchedTx(matchedTx) {
			continue
		}

		tradePairs, err := getSortedTradePairsFromMatchedTx(matchedTx)
		if err != nil {
			return err
		}

		actualMatchedTx, err := matchEngine.NextMatchedTx(tradePairs...)
		if err != nil {
			return err
		}

		if len(matchedTx.Inputs) != len(actualMatchedTx.Inputs) {
			return errors.New("length of matched tx input is not equals to actual matched tx input")
		}

		spendOutputIDs := make(map[string]bool)
		for _, input := range matchedTx.Inputs {
			spendOutputID, err := input.SpentOutputID()
			if err != nil {
				return err
			}

			spendOutputIDs[spendOutputID.String()] = true
		}

		for _, input := range actualMatchedTx.Inputs {
			spendOutputID, err := input.SpentOutputID()
			if err != nil {
				return err
			}

			if _, ok := spendOutputIDs[spendOutputID.String()]; !ok {
				return errors.New("spend output id of matched tx is not equals to actual matched tx")
			}
		}
	}
	return nil
}

func getSortedTradePairsFromMatchedTx(tx *types.Tx) ([]*common.TradePair, error) {
	assetMap := make(map[bc.AssetID]bc.AssetID)
	var firstTradePair *common.TradePair
	for _, tx := range tx.Inputs {
		contractArgs, err := segwit.DecodeP2WMCProgram(tx.ControlProgram())
		if err != nil {
			return nil, err
		}

		assetMap[tx.AssetID()] = contractArgs.RequestedAsset
		if firstTradePair == nil {
			firstTradePair = &common.TradePair{FromAssetID: tx.AssetAmount().AssetId, ToAssetID: &contractArgs.RequestedAsset}
		}
	}

	tradePairs := []*common.TradePair{firstTradePair}
	for tradePair := firstTradePair; *tradePair.ToAssetID != *firstTradePair.FromAssetID; {
		nextTradePairToAssetID, ok := assetMap[*tradePair.ToAssetID]
		if !ok {
			return nil, errInvalidTradePairs
		}

		tradePair = &common.TradePair{FromAssetID: tradePair.ToAssetID, ToAssetID: &nextTradePairToAssetID}
		tradePairs = append(tradePairs, tradePair)
	}

	if len(tradePairs) != len(tx.Inputs) {
		return nil, errInvalidTradePairs
	}
	return tradePairs, nil
}

// DetachBlock parse pending order and cancel from the the transactions of block
// and add cancel order to the dex db, remove pending order from dex db.
func (m *MovCore) DetachBlock(block *types.Block) error {
	deleteOrders, addOrders, err := applyTransactions(block.Transactions)
	if err != nil {
		return err
	}

	return m.movStore.ProcessOrders(addOrders, deleteOrders, &block.BlockHeader)
}

// BeforeProposalBlock return all transactions than can be matched, and the number of transactions cannot exceed the given capacity.
func (m *MovCore) BeforeProposalBlock(capacity int, nodeProgram []byte) ([]*types.Tx, error) {
	matchEngine := match.NewEngine(m.movStore, maxFeeRate, nodeProgram)
	tradePairMap := make(map[string]bool)
	tradePairIterator := database.NewTradePairIterator(m.movStore)

	var packagedTxs []*types.Tx
	for len(packagedTxs) < capacity && tradePairIterator.HasNext() {
		tradePair := tradePairIterator.Next()
		if tradePairMap[tradePair.Key()] {
			continue
		}
		tradePairMap[tradePair.Key()] = true
		tradePairMap[tradePair.Reverse().Key()] = true

		for len(packagedTxs) < capacity && matchEngine.HasMatchedTx(tradePair, tradePair.Reverse()) {
			matchedTx, err := matchEngine.NextMatchedTx(tradePair, tradePair.Reverse())
			if err != nil {
				return nil, err
			}

			packagedTxs = append(packagedTxs, matchedTx)
		}
	}
	return packagedTxs, nil
}

func (m *MovCore) IsDust(tx *types.Tx) bool {
	for _, input := range tx.Inputs {
		if segwit.IsP2WMCScript(input.ControlProgram()) && !contract.IsCancelClauseSelector(input) {
			return true
		}
	}
	return false
}

func applyTransactions(txs []*types.Tx) ([]*common.Order, []*common.Order, error) {
	deleteOrderMap := make(map[string]*common.Order)
	addOrderMap := make(map[string]*common.Order)
	for _, tx := range txs {
		addOrders, err := getAddOrdersFromTx(tx)
		if err != nil {
			return nil, nil, err
		}

		for _, order := range addOrders {
			addOrderMap[order.Key()] = order
		}

		deleteOrders, err := getDeleteOrdersFromTx(tx)
		if err != nil {
			return nil, nil, err
		}

		for _, order := range deleteOrders {
			deleteOrderMap[order.Key()] = order
		}
	}

	addOrders, deleteOrders := mergeOrders(addOrderMap, deleteOrderMap)
	return addOrders, deleteOrders, nil
}

func mergeOrders(addOrderMap, deleteOrderMap map[string]*common.Order) ([]*common.Order, []*common.Order) {
	var deleteOrders, addOrders []*common.Order
	for orderID, order := range addOrderMap {
		if _, ok := deleteOrderMap[orderID]; ok {
			delete(deleteOrderMap, orderID)
			continue
		}
		addOrders = append(addOrders, order)
	}

	for _, order := range deleteOrderMap {
		deleteOrders = append(deleteOrders, order)
	}
	return addOrders, deleteOrders
}

func getAddOrdersFromTx(tx *types.Tx) ([]*common.Order, error) {
	var orders []*common.Order
	for i, output := range tx.Outputs {
		if output.OutputType() != types.IntraChainOutputType || !segwit.IsP2WMCScript(output.ControlProgram()) {
			continue
		}

		order, err := common.NewOrderFromOutput(tx, i)
		if err != nil {
			return nil, err
		}

		orders = append(orders, order)
	}
	return orders, nil
}

func getDeleteOrdersFromTx(tx *types.Tx) ([]*common.Order, error) {
	var orders []*common.Order
	for i, input := range tx.Inputs {
		if input.InputType() != types.SpendInputType || !segwit.IsP2WMCScript(input.ControlProgram()) {
			continue
		}

		order, err := common.NewOrderFromInput(tx, i)
		if err != nil {
			return nil, err
		}

		orders = append(orders, order)
	}
	return orders, nil
}
