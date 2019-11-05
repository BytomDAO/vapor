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
	errInvalidTradePairs             = errors.New("The trade pairs in the tx input is invalid")
	errStatusFailMustFalse           = errors.New("status fail of transaction does not allow to be true")
	errInputProgramMustP2WMCScript   = errors.New("input program of matched tx must p2wmc script")
	errExistCancelOrderInMatchedTx   = errors.New("can't exist cancel order in the matched transaction")
	errExistTradeInCancelOrderTx     = errors.New("can't exist trade in the cancel order transaction")
	errAmountOfFeeGreaterThanMaximum = errors.New("amount of fee greater than max fee amount")
	errAssetIDMustUniqueInMatchedTx  = errors.New("asset id must unique in matched transaction")
	errRatioOfTradeLessThanZero      = errors.New("ratio arguments must greater than zero")
	errNumeratorOfRatioIsOverflow    = errors.New("ratio numerator of contract args product input amount is overflow")
	errLengthOfInputIsIncorrect      = errors.New("length of matched tx input is not equals to actual matched tx input")
	errSpendOutputIDIsIncorrect      = errors.New("spend output id of matched tx is not equals to actual matched tx")
)

// MovCore represent the core logic of the match module, which include generate match transactions before packing the block,
// verify the match transaction in block is correct, and update the order table according to the transaction.
type MovCore struct {
	movStore         database.MovStore
	startBlockHeight uint64
}

// NewMovCore return a instance of MovCore by path of mov db
func NewMovCore(dbBackend, dbDir string, startBlockHeight uint64) *MovCore {
	movDB := dbm.NewDB("mov", dbBackend, dbDir)
	return &MovCore{movStore: database.NewLevelDBMovStore(movDB), startBlockHeight: startBlockHeight}
}

// Name return the name of current module
func (m *MovCore) Name() string {
	return "MOV"
}

// ChainStatus return the current block height and block hash in dex core
func (m *MovCore) ChainStatus() (uint64, *bc.Hash, error) {
	state, err := m.movStore.GetMovDatabaseState()
	if err != nil {
		return 0, nil, err
	}

	return state.Height, state.Hash, nil
}

// ValidateBlock no need to verify the block header, becaure the first module has been verified.
// just need to verify the transactions in the block.
func (m *MovCore) ValidateBlock(block *types.Block, verifyResults []*bc.TxVerifyResult) error {
	return m.ValidateTxs(block.Transactions, verifyResults)
}

// ValidateTxs validate the trade transaction.
func (m *MovCore) ValidateTxs(txs []*types.Tx, verifyResults []*bc.TxVerifyResult) error {
	for i, tx := range txs {
		if err := m.ValidateTx(tx, verifyResults[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *MovCore) ValidateTx(tx *types.Tx, verifyResult *bc.TxVerifyResult) error {
	if common.IsMatchedTx(tx) {
		if err := validateMatchedTx(tx, verifyResult); err != nil {
			return err
		}
	}

	if common.IsCancelOrderTx(tx) {
		if err := validateCancelOrderTx(tx, verifyResult); err != nil {
			return err
		}
	}

	for _, output := range tx.Outputs {
		if !segwit.IsP2WMCScript(output.ControlProgram()) {
			continue
		}
		if verifyResult.StatusFail {
			return errStatusFailMustFalse
		}

		if err := validateMagneticContractArgs(output.AssetAmount().Amount, output.ControlProgram()); err != nil {
			return err
		}
	}
	return nil
}

func validateMatchedTx(tx *types.Tx, verifyResult *bc.TxVerifyResult) error {
	if verifyResult.StatusFail {
		return errStatusFailMustFalse
	}

	fromAssetIDMap := make(map[string]bool)
	toAssetIDMap := make(map[string]bool)
	for i, input := range tx.Inputs {
		if !segwit.IsP2WMCScript(input.ControlProgram()) {
			return errInputProgramMustP2WMCScript
		}

		if contract.IsCancelClauseSelector(input) {
			return errExistCancelOrderInMatchedTx
		}

		order, err := common.NewOrderFromInput(tx, i)
		if err != nil {
			return err
		}

		fromAssetIDMap[order.FromAssetID.String()] = true
		toAssetIDMap[order.ToAssetID.String()] = true
	}

	if len(fromAssetIDMap) != len(tx.Inputs) || len(toAssetIDMap) != len(tx.Inputs) {
		return errAssetIDMustUniqueInMatchedTx
	}

	return validateMatchedTxFeeAmount(tx)
}

func validateCancelOrderTx(tx *types.Tx, verifyResult *bc.TxVerifyResult) error {
	if verifyResult.StatusFail {
		return errStatusFailMustFalse
	}

	for _, input := range tx.Inputs {
		if !segwit.IsP2WMCScript(input.ControlProgram()) {
			return errInputProgramMustP2WMCScript
		}

		if contract.IsTradeClauseSelector(input) {
			return errExistTradeInCancelOrderTx
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
			return errAmountOfFeeGreaterThanMaximum
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
		return errRatioOfTradeLessThanZero
	}

	if _, ok := checked.MulInt64(int64(inputAmount), contractArgs.RatioNumerator); !ok {
		return errNumeratorOfRatioIsOverflow
	}
	return nil
}

// ApplyBlock parse pending order and cancel from the the transactions of block
// and add pending order to the dex db, remove cancel order from dex db.
func (m *MovCore) ApplyBlock(block *types.Block) error {
	if block.Height < m.startBlockHeight {
		return nil
	}

	if block.Height == m.startBlockHeight {
		blockHash := block.Hash()
		if err := m.movStore.InitDBState(block.Height, &blockHash); err != nil {
			return err
		}

		return nil
	}

	if err := m.validateMatchedTxSequence(block.Transactions); err != nil {
		return err
	}

	addOrders, deleteOrders, err := applyTransactions(block.Transactions)
	if err != nil {
		return err
	}

	return m.movStore.ProcessOrders(addOrders, deleteOrders, &block.BlockHeader)
}

func (m *MovCore) validateMatchedTxSequence(txs []*types.Tx) error {
	orderTable, err := buildOrderTable(m.movStore, txs)
	if err != nil {
		return err
	}

	matchEngine := match.NewEngine(orderTable, maxFeeRate, nil)
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
			return errLengthOfInputIsIncorrect
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
				return errSpendOutputIDIsIncorrect
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
	if block.Height <= m.startBlockHeight {
		return nil
	}

	deleteOrders, addOrders, err := applyTransactions(block.Transactions)
	if err != nil {
		return err
	}

	return m.movStore.ProcessOrders(addOrders, deleteOrders, &block.BlockHeader)
}

// BeforeProposalBlock return all transactions than can be matched, and the number of transactions cannot exceed the given capacity.
func (m *MovCore) BeforeProposalBlock(txs []*types.Tx, nodeProgram []byte, blockHeight uint64, gasLeft int64) ([]*types.Tx, int64, error) {
	if blockHeight <= m.startBlockHeight {
		return nil, 0, nil
	}

	orderTable, err := buildOrderTable(m.movStore, txs)
	if err != nil {
		return nil, 0, err
	}

	matchEngine := match.NewEngine(orderTable, maxFeeRate, nodeProgram)
	tradePairMap := make(map[string]bool)
	tradePairIterator := database.NewTradePairIterator(m.movStore)

	var packagedTxs []*types.Tx
	for gasLeft > 0 && tradePairIterator.HasNext() {
		tradePair := tradePairIterator.Next()
		if tradePairMap[tradePair.Key()] {
			continue
		}
		tradePairMap[tradePair.Key()] = true
		tradePairMap[tradePair.Reverse().Key()] = true

		for gasLeft > 0 && matchEngine.HasMatchedTx(tradePair, tradePair.Reverse()) {
			matchedTx, err := matchEngine.NextMatchedTx(tradePair, tradePair.Reverse())
			if err != nil {
				return nil, 0, err
			}

			gasUsed := calcMatchedTxGasUsed(matchedTx)
			if gasLeft-gasUsed >= 0 {
				packagedTxs = append(packagedTxs, matchedTx)
			}
			gasLeft -= gasUsed
		}
	}
	return packagedTxs, gasLeft, nil
}

func calcMatchedTxGasUsed(tx *types.Tx) int64 {
	return int64(len(tx.Inputs))*150 + int64(tx.SerializedSize)
}

func buildOrderTable(store database.MovStore, txs []*types.Tx) (*match.OrderTable, error) {
	var nonMatchedTxs []*types.Tx
	for _, tx := range txs {
		if !common.IsMatchedTx(tx) {
			nonMatchedTxs = append(nonMatchedTxs, tx)
		}
	}

	var arrivalAddOrders, arrivalDelOrders []*common.Order
	for _, tx := range nonMatchedTxs {
		addOrders, err := getAddOrdersFromTx(tx)
		if err != nil {
			return nil, err
		}

		delOrders, err := getDeleteOrdersFromTx(tx)
		if err != nil {
			return nil, err
		}

		arrivalAddOrders = append(arrivalAddOrders, addOrders...)
		arrivalDelOrders = append(arrivalDelOrders, delOrders...)
	}

	return match.NewOrderTable(store, arrivalAddOrders, arrivalDelOrders), nil
}

// IsDust block the transaction that are not generated by the match engine
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
