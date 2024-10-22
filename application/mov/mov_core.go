package mov

import (
	"encoding/hex"

	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/application/mov/contract"
	"github.com/bytom/vapor/application/mov/database"
	"github.com/bytom/vapor/application/mov/match"
	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/consensus/segwit"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
)

var (
	errChainStatusHasAlreadyInit    = errors.New("mov chain status has already initialized")
	errInvalidTradePairs            = errors.New("The trade pairs in the tx input is invalid")
	errStatusFailMustFalse          = errors.New("status fail of transaction does not allow to be true")
	errInputProgramMustP2WMCScript  = errors.New("input program of trade tx must p2wmc script")
	errExistCancelOrderInMatchedTx  = errors.New("can't exist cancel order in the matched transaction")
	errExistTradeInCancelOrderTx    = errors.New("can't exist trade in the cancel order transaction")
	errAssetIDMustUniqueInMatchedTx = errors.New("asset id must unique in matched transaction")
	errRatioOfTradeLessThanZero     = errors.New("ratio arguments must greater than zero")
	errSpendOutputIDIsIncorrect     = errors.New("spend output id of matched tx is not equals to actual matched tx")
	errRequestAmountMath            = errors.New("request amount of order less than one or big than max of int64")
	errNotMatchedOrder              = errors.New("order in matched tx is not matched")
	errNotConfiguredRewardProgram   = errors.New("reward program is not configured properly")
	errRewardProgramIsWrong         = errors.New("the reward program is not correct")
)

// Core represent the core logic of the match module, which include generate match transactions before packing the block,
// verify the match transaction in block is correct, and update the order table according to the transaction.
type Core struct {
	movStore         database.MovStore
	startBlockHeight uint64
}

// NewCore return a instance of Core by path of mov db
func NewCore(dbBackend, dbDir string, startBlockHeight uint64) *Core {
	movDB := dbm.NewDB("mov", dbBackend, dbDir)
	return &Core{movStore: database.NewLevelDBMovStore(movDB), startBlockHeight: startBlockHeight}
}

// NewCoreWithDB return a instance of Core by movStore
func NewCoreWithDB(store *database.LevelDBMovStore, startBlockHeight uint64) *Core {
	return &Core{movStore: store, startBlockHeight: startBlockHeight}
}

// ApplyBlock parse pending order and cancel from the the transactions of block
// and add pending order to the dex db, remove cancel order from dex db.
func (m *Core) ApplyBlock(block *types.Block) error {
	if block.Height < m.startBlockHeight {
		return nil
	}

	if block.Height == m.startBlockHeight {
		blockHash := block.Hash()
		return m.InitChainStatus(&blockHash)
	}

	if err := m.validateMatchedTxSequence(movTxs(block)); err != nil {
		return err
	}

	addOrders, deleteOrders, err := decodeTxsOrders(movTxs(block))
	if err != nil {
		return err
	}

	return m.movStore.ProcessOrders(addOrders, deleteOrders, &block.BlockHeader)
}

// Tx contains raw transaction and the sequence of tx in block
type Tx struct {
	rawTx       *types.Tx
	blockHeight uint64
	txIndex     uint64
}

// NewTx create a new Tx instance
func NewTx(tx *types.Tx, blockHeight, txIndex uint64) *Tx {
	return &Tx{rawTx: tx, blockHeight: blockHeight, txIndex: txIndex}
}

// BeforeProposalBlock return all transactions than can be matched, and the number of transactions cannot exceed the given capacity.
func (m *Core) BeforeProposalBlock(block *types.Block, gasLeft int64, isTimeout func() bool) ([]*types.Tx, error) {
	if block.Height <= m.startBlockHeight {
		return nil, nil
	}

	orderBook, err := buildOrderBook(m.movStore, movTxs(block))
	if err != nil {
		return nil, err
	}

	program, _ := getRewardProgram(block.Height)
	rewardProgram, err := hex.DecodeString(program)
	if err != nil {
		return nil, errNotConfiguredRewardProgram
	}

	matchEngine := match.NewEngine(orderBook, rewardProgram)
	tradePairIterator := database.NewTradePairIterator(m.movStore)
	matchCollector := newMatchTxCollector(matchEngine, tradePairIterator, gasLeft, isTimeout)
	return matchCollector.result()
}

// ChainStatus return the current block height and block hash in dex core
func (m *Core) ChainStatus() (uint64, *bc.Hash, error) {
	state, err := m.movStore.GetMovDatabaseState()
	if err == database.ErrNotInitDBState {
		return 0, nil, protocol.ErrNotInitSubProtocolChainStatus
	}

	if err != nil {
		return 0, nil, err
	}

	return state.Height, state.Hash, nil
}

// DetachBlock parse pending order and cancel from the the transactions of block
// and add cancel order to the dex db, remove pending order from dex db.
func (m *Core) DetachBlock(block *types.Block) error {
	if block.Height < m.startBlockHeight {
		return nil
	}

	if block.Height == m.startBlockHeight {
		m.movStore.Clear()
		return nil
	}

	deleteOrders, addOrders, err := decodeTxsOrders(movTxs(block))
	if err != nil {
		return err
	}

	return m.movStore.ProcessOrders(addOrders, deleteOrders, &block.BlockHeader)
}

// InitChainStatus used to init the start block height and start block hash to store
func (m *Core) InitChainStatus(startHash *bc.Hash) error {
	if _, err := m.movStore.GetMovDatabaseState(); err == nil {
		return errChainStatusHasAlreadyInit
	}

	return m.movStore.InitDBState(m.startBlockHeight, startHash)
}

// IsDust block the transaction that are not generated by the match engine
func (m *Core) IsDust(tx *types.Tx) bool {
	for _, input := range tx.Inputs {
		if segwit.IsP2WMCScript(input.ControlProgram()) && !contract.IsCancelClauseSelector(input) {
			return true
		}
	}
	return false
}

// Name return the name of current module
func (m *Core) Name() string {
	return "MOV"
}

// StartHeight return the start block height of current module
func (m *Core) StartHeight() uint64 {
	return m.startBlockHeight
}

// ValidateBlock no need to verify the block header, because the first module has been verified.
// just need to verify the transactions in the block.
func (m *Core) ValidateBlock(block *types.Block, verifyResults []*bc.TxVerifyResult) error {
	for i, tx := range block.Transactions {
		if err := m.ValidateTx(tx, verifyResults[i], block.Height); err != nil {
			return err
		}
	}
	return nil
}

// ValidateTx validate one transaction.
func (m *Core) ValidateTx(tx *types.Tx, verifyResult *bc.TxVerifyResult, blockHeight uint64) error {
	if blockHeight <= m.startBlockHeight {
		return nil
	}

	if verifyResult.StatusFail {
		return errStatusFailMustFalse
	}

	if common.IsMatchedTx(tx) {
		if err := validateMatchedTx(tx, blockHeight); err != nil {
			return err
		}
	} else if common.IsCancelOrderTx(tx) {
		if err := validateCancelOrderTx(tx); err != nil {
			return err
		}
	}

	for _, output := range tx.Outputs {
		if !segwit.IsP2WMCScript(output.ControlProgram()) {
			continue
		}

		if err := validateMagneticContractArgs(output.AssetAmount(), output.ControlProgram()); err != nil {
			return err
		}
	}
	return nil
}

// matchedTxFee is object to record the mov tx's fee information
type matchedTxFee struct {
	rewardProgram []byte
	amount        uint64
}

// calcFeeAmount return the amount of fee in the matching transaction
func calcFeeAmount(matchedTx *types.Tx) (map[bc.AssetID]*matchedTxFee, error) {
	assetFeeMap := make(map[bc.AssetID]*matchedTxFee)
	dealProgMaps := make(map[string]bool)

	for _, input := range matchedTx.Inputs {
		assetFeeMap[input.AssetID()] = &matchedTxFee{amount: input.AssetAmount().Amount}
		contractArgs, err := segwit.DecodeP2WMCProgram(input.ControlProgram())
		if err != nil {
			return nil, err
		}

		dealProgMaps[hex.EncodeToString(contractArgs.SellerProgram)] = true
	}

	for _, output := range matchedTx.Outputs {
		assetAmount := output.AssetAmount()
		if _, ok := dealProgMaps[hex.EncodeToString(output.ControlProgram())]; ok || segwit.IsP2WMCScript(output.ControlProgram()) {
			assetFeeMap[*assetAmount.AssetId].amount -= assetAmount.Amount
			if assetFeeMap[*assetAmount.AssetId].amount <= 0 {
				delete(assetFeeMap, *assetAmount.AssetId)
			}
		} else if assetFeeMap[*assetAmount.AssetId].rewardProgram == nil {
			assetFeeMap[*assetAmount.AssetId].rewardProgram = output.ControlProgram()
		} else {
			return nil, errors.Wrap(errRewardProgramIsWrong, "double reward program")
		}
	}
	return assetFeeMap, nil
}

func validateCancelOrderTx(tx *types.Tx) error {
	for _, input := range tx.Inputs {
		if segwit.IsP2WMCScript(input.ControlProgram()) && !contract.IsCancelClauseSelector(input) {
			return errInputProgramMustP2WMCScript
		}
	}
	return nil
}

func validateMagneticContractArgs(fromAssetAmount bc.AssetAmount, program []byte) error {
	contractArgs, err := segwit.DecodeP2WMCProgram(program)
	if err != nil {
		return err
	}

	if *fromAssetAmount.AssetId == contractArgs.RequestedAsset {
		return errInvalidTradePairs
	}

	if contractArgs.RatioNumerator <= 0 || contractArgs.RatioDenominator <= 0 {
		return errRatioOfTradeLessThanZero
	}

	if match.CalcRequestAmount(fromAssetAmount.Amount, contractArgs.RatioNumerator, contractArgs.RatioDenominator) < 1 {
		return errRequestAmountMath
	}
	return nil
}

func validateMatchedTx(tx *types.Tx, blockHeight uint64) error {
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

	if inputSize := len(tx.Inputs); len(fromAssetIDMap) != inputSize || len(toAssetIDMap) != inputSize {
		return errAssetIDMustUniqueInMatchedTx
	}

	return validateMatchedTxFee(tx, blockHeight)
}

func validateMatchedTxFee(tx *types.Tx, blockHeight uint64) error {
	matchedTxFees, err := calcFeeAmount(tx)
	if err != nil {
		return err
	}

	for _, fee := range matchedTxFees {
		if err := validateRewardProgram(blockHeight, hex.EncodeToString(fee.rewardProgram)); err != nil {
			return err
		}
	}

	orders, err := parseDeleteOrdersFromTx(tx)
	if err != nil {
		return err
	}

	receivedAmount, priceDiffs := match.CalcReceivedAmount(orders)
	feeAmounts := make(map[bc.AssetID]uint64)
	for assetID, fee := range matchedTxFees {
		feeAmounts[assetID] = fee.amount
	}

	feeStrategy := match.NewDefaultFeeStrategy()
	return feeStrategy.Validate(receivedAmount, priceDiffs, feeAmounts, blockHeight)
}

func (m *Core) validateMatchedTxSequence(txs []*Tx) error {
	orderBook := match.NewOrderBook(m.movStore, nil, nil)
	for _, tx := range txs {
		if common.IsMatchedTx(tx.rawTx) {
			tradePairs, err := parseTradePairsFromMatchedTx(tx.rawTx)
			if err != nil {
				return err
			}

			orders := orderBook.PeekOrders(tradePairs)
			if err := validateSpendOrders(tx.rawTx, orders); err != nil {
				return err
			}

			orderBook.PopOrders(tradePairs)
		} else if common.IsCancelOrderTx(tx.rawTx) {
			orders, err := parseDeleteOrdersFromTx(tx.rawTx)
			if err != nil {
				return err
			}

			for _, order := range orders {
				orderBook.DelOrder(order)
			}
		}

		addOrders, err := parseAddOrdersFromTx(tx)
		if err != nil {
			return err
		}

		for _, order := range addOrders {
			orderBook.AddOrder(order)
		}
	}
	return nil
}

func validateSpendOrders(tx *types.Tx, orders []*common.Order) error {
	if len(tx.Inputs) != len(orders) {
		return errNotMatchedOrder
	}

	spendOutputIDs := make(map[string]bool)
	for _, input := range tx.Inputs {
		spendOutputID, err := input.SpentOutputID()
		if err != nil {
			return err
		}

		spendOutputIDs[spendOutputID.String()] = true
	}

	for _, order := range orders {
		outputID := order.UTXOHash().String()
		if _, ok := spendOutputIDs[outputID]; !ok {
			return errSpendOutputIDIsIncorrect
		}
	}
	return nil
}

func decodeTxsOrders(txs []*Tx) ([]*common.Order, []*common.Order, error) {
	deleteOrderMap := make(map[string]*common.Order)
	addOrderMap := make(map[string]*common.Order)
	for _, tx := range txs {
		addOrders, err := parseAddOrdersFromTx(tx)
		if err != nil {
			return nil, nil, err
		}

		for _, order := range addOrders {
			addOrderMap[order.Key()] = order
		}

		deleteOrders, err := parseDeleteOrdersFromTx(tx.rawTx)
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

func buildOrderBook(store database.MovStore, txs []*Tx) (*match.OrderBook, error) {
	var arrivalAddOrders, arrivalDelOrders []*common.Order
	for _, tx := range txs {
		addOrders, err := parseAddOrdersFromTx(tx)
		if err != nil {
			return nil, err
		}

		delOrders, err := parseDeleteOrdersFromTx(tx.rawTx)
		if err != nil {
			return nil, err
		}

		arrivalAddOrders = append(arrivalAddOrders, addOrders...)
		arrivalDelOrders = append(arrivalDelOrders, delOrders...)
	}

	return match.NewOrderBook(store, arrivalAddOrders, arrivalDelOrders), nil
}

func parseAddOrdersFromTx(tx *Tx) ([]*common.Order, error) {
	var orders []*common.Order
	for i, output := range tx.rawTx.Outputs {
		if output.OutputType() != types.IntraChainOutputType || !segwit.IsP2WMCScript(output.ControlProgram()) {
			continue
		}

		if output.AssetAmount().Amount == 0 {
			continue
		}

		order, err := common.NewOrderFromOutput(tx.rawTx, i, tx.blockHeight, tx.txIndex)
		if err != nil {
			return nil, err
		}

		orders = append(orders, order)
	}
	return orders, nil
}

func parseDeleteOrdersFromTx(tx *types.Tx) ([]*common.Order, error) {
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

func parseTradePairsFromMatchedTx(tx *types.Tx) ([]*common.TradePair, error) {
	var tradePairs []*common.TradePair
	for _, tx := range tx.Inputs {
		contractArgs, err := segwit.DecodeP2WMCProgram(tx.ControlProgram())
		if err != nil {
			return nil, err
		}

		tradePairs = append(tradePairs, &common.TradePair{FromAssetID: tx.AssetAmount().AssetId, ToAssetID: &contractArgs.RequestedAsset})
	}
	return tradePairs, nil
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

func movTxs(block *types.Block) []*Tx {
	var movTxs []*Tx
	for i, tx := range block.Transactions {
		movTxs = append(movTxs, NewTx(tx, block.Height, uint64(i)))
	}
	return movTxs
}

// getRewardProgram return the reward program by specified block height
// if no reward program configured, then will return empty string
// if reward program of 0-100 height is configured, but the specified height is 200, then will return  0-100's reward program
// the second return value represent whether to find exactly
func getRewardProgram(height uint64) (string, bool) {
	rewardPrograms := consensus.ActiveNetParams.MovRewardPrograms
	if len(rewardPrograms) == 0 {
		return "51", false
	}

	var program string
	for _, rewardProgram := range rewardPrograms {
		program = rewardProgram.Program
		if height >= rewardProgram.BeginBlock && height <= rewardProgram.EndBlock {
			return program, true
		}
	}
	return program, false
}

func validateRewardProgram(height uint64, program string) error {
	rewardProgram, exact := getRewardProgram(height)
	if exact && rewardProgram != program {
		return errRewardProgramIsWrong
	}
	return nil
}
