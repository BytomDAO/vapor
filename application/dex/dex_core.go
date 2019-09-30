package dex

import (
	"fmt"
	"encoding/hex"

	"github.com/vapor/errors"
	"github.com/vapor/application/dex/common"
	"github.com/vapor/application/dex/database"
	"github.com/vapor/application/dex/match"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm"
)

type DexCore struct {
	dexDB       *database.DexTradeOrderDB
	matchEngine *match.MatchEngine
}

// ChainStatus return the current block hegiht and block hash in dex core
func (d *DexCore) ChainStatus() (uint64, *bc.Hash, error) {
	state, err := d.dexDB.GetDexDatabaseState()
	if err != nil {
		return 0, nil, err
	}

	return state.Height, state.Hash, nil
}

func (d *DexCore) ValidateBlock(block *types.Block) error {
	if err := d.ValidateTxs(block.Transactions); err != nil {
		return err
	}
	return nil
}

// ValidateTxs validate the matched transactions is generated accroding to the matching rule.
func (d *DexCore) ValidateTxs(txs []*types.Tx) error {
	if err := d.validateMatchedTxs(txs); err != nil {
		return err
	}
	return nil
}

func (d *DexCore) validateMatchedTxs(txs []*types.Tx) error {
	hasMatchedTxMap := make(map[string]bool)
	for _, tx := range txs {
		if isMatchedTx(tx) {
			hasMatchedTxMap[tx.ID.String()] = true
		}
	}

	orderTables, err := d.generateOrderTables(txs)
	if err != nil {
		return err
	}

	for _, orderTable := range orderTables {
		matchedTxs, err := d.matchEngine.GenerateMatchedTxs(orderTable)
		if err != nil {
			return err
		}

		for _, matchedTx := range matchedTxs {
			if hasMatchedTxMap[matchedTx.ID.String()] {
				delete(hasMatchedTxMap, matchedTx.ID.String())
			}
			if len(hasMatchedTxMap) == 0 {
				return nil
			}
		}
	}
	return errors.New("fail to validate match transaction")
}

// ApplyBlock parse pending order and cancel from the the transactions of block
// and add pending order to the dex db, remove cancel order from dex db.
func (d *DexCore) ApplyBlock(block *types.Block) error {
	pendingOrders, cancelOrders, err := applyTransactions(block.Transactions)
	if err != nil {
		return err
	}

	blockHash := block.Hash()
	return d.dexDB.ProcessOrders(pendingOrders, cancelOrders, block.Height, &blockHash)
}

// DetachBlock parse pending order and cancel from the the transactions of block
// and add cancel order to the dex db, remove pending order from dex db.
func (d *DexCore) DetachBlock(block *types.Block) error {
	pendingOrders, cancelOrders, err := applyTransactions(block.Transactions)
	if err != nil {
		return err
	}

	return d.dexDB.ProcessOrders(cancelOrders, pendingOrders, block.Height-1, &block.PreviousBlockHash)
}

// BeforeProposalBlock get all pending orders from the dex db, parse pending orders and cancel orders from transactions
// Then merge the two, use match engine to generate matched transactions, finally return them.
func (d *DexCore) BeforeProposalBlock(txs []*types.Tx, numOfPackage int) ([]*types.Tx, error) {
	orderTables, err := d.generateOrderTables(txs)
	if err != nil {
		return nil, err
	}

	var packagedTxs []*types.Tx
	for _, orderTable := range orderTables {
		matchedTxs, err := d.matchEngine.GenerateMatchedTxs(orderTable)
		if err != nil {
			return nil, err
		}

		num := len(matchedTxs)
		if len(packagedTxs)+len(matchedTxs) > numOfPackage {
			num = numOfPackage - len(packagedTxs)
		}
		for i := 0; i < num; i++ {
			packagedTxs = append(packagedTxs, matchedTxs[i])
		}
	}
	return packagedTxs, nil
}

func (d *DexCore) IsDust(tx *types.Tx) bool {
	return false
}

func (d *DexCore) generateOrderTables(txs []*types.Tx) ([]*match.OrderTable, error) {
	tradePairs, err := d.dexDB.GetTradePairsWithStart(nil)
	if err != nil {
		return nil, err
	}

	var orderTables []*match.OrderTable
	orderMap := make(map[string][]*common.Order)
	for _, tradePair := range tradePairs {
		if _, ok := orderMap[tradePair.FromAssetID.String()+tradePair.ToAssetID.String()]; ok {
			continue
		}

		buyOrders, err := d.dexDB.ListOrders(tradePair.FromAssetID.String(), tradePair.ToAssetID.String(), 0)
		if err != nil {
			return nil, err
		}

		sellOrders, err := d.dexDB.ListOrders(tradePair.ToAssetID.String(), tradePair.FromAssetID.String(), 0)
		if err != nil {
			return nil, err
		}

		orderTables = append(orderTables, &match.OrderTable{BuyOrders: buyOrders, SellOrders: sellOrders})
		orderMap[tradePair.FromAssetID.String()+tradePair.ToAssetID.String()] = buyOrders
		orderMap[tradePair.ToAssetID.String()+tradePair.FromAssetID.String()] = sellOrders
	}
	return orderTables, nil
}

func applyTransactions(txs []*types.Tx) ([]*common.Order, []*common.Order, error) {
	var pendingOrders []*common.Order
	var cancelOrders []*common.Order
	var matchedTxs []*types.Tx
	for _, tx := range txs {
		subPendingOrders, err := getPendingOrderIfPresent(tx)
		if err != nil {
			return nil, nil, err
		}

		pendingOrders = append(pendingOrders, subPendingOrders...)

		subCancelOrders, err := getCancelOrderIfPresent(tx)
		if err != nil {
			return nil, nil, err
		}

		cancelOrders = append(cancelOrders, subCancelOrders...)

		if isMatchedTx(tx) {
			matchedTxs = append(matchedTxs, tx)
		}
	}
	subPendingOrders, subCancelOrders, err := applyMatchedTxs(matchedTxs)
	if err != nil {
		return nil, nil, nil
	}

	pendingOrders = append(pendingOrders, subPendingOrders...)
	cancelOrders = append(cancelOrders, subCancelOrders...)
	return pendingOrders, cancelOrders, nil
}

func applyMatchedTxs(txs []*types.Tx) ([]*common.Order, []*common.Order, error) {
	cancelOrderMap := make(map[string]*common.Order)
	pendingOrderMap := make(map[string]*common.Order)
	for _, tx := range txs {
		tradeOrders, err := getTradeOrderIfPresent(tx)
		if err != nil {
			return nil, nil, err
		}

		for _, order := range tradeOrders {
			orderID := fmt.Sprintf("%s:%d", order.Utxo.SourceID, order.Utxo.SourcePos)
			cancelOrderMap[orderID] = order
		}

		pendingOrders, err := getPendingOrderIfPresent(tx)
		if err != nil {
			return nil, nil, err
		}

		for _, order := range pendingOrders {
			orderID := fmt.Sprintf("%s:%d", order.Utxo.SourceID, order.Utxo.SourcePos)
			pendingOrderMap[orderID] = order
		}
	}

	var cancelOrders, pendingOrders []*common.Order
	for orderID, order := range pendingOrderMap {
		if cancelOrderMap[orderID] != nil {
			delete(cancelOrderMap, orderID)
			continue
		}
		pendingOrders = append(pendingOrders, order)
	}
	for _, order := range cancelOrders {
		cancelOrders = append(cancelOrders, order)
	}
	return pendingOrders, cancelOrders, nil
}

func getPendingOrderIfPresent(tx *types.Tx) ([]*common.Order, error) {
	var orders []*common.Order
	for i, output := range tx.Outputs {
		if output.OutputType() == types.IntraChainOutputType && IsP2WMCScript(output.ControlProgram()) {
			order, err := common.OutputToOrder(tx, i)
			if err != nil {
				return nil, err
			}

			orders = append(orders, order)
		}
	}
	return orders, nil
}

func getTradeOrderIfPresent(tx *types.Tx) ([]*common.Order, error) {
	return getInputOrderByClauseSelector(tx, isTradeClauseSelector)
}

func getCancelOrderIfPresent(tx *types.Tx) ([]*common.Order, error) {
	return getInputOrderByClauseSelector(tx, isCancelClauseSelector)
}

func getInputOrderByClauseSelector(tx *types.Tx, checkClauseSelector func(*types.TxInput) bool) ([]*common.Order, error) {
	var orders []*common.Order
	for _, input := range tx.Inputs {
		if input.InputType() != types.SpendInputType || IsP2WMCScript(input.ControlProgram()) {
			continue
		}

		if checkClauseSelector(input) {
			order, err := common.InputToOrder(input)
			if err != nil {
				return nil, err
			}

			orders = append(orders, order)
		}
	}
	return orders, nil
}

func isMatchedTx(tx *types.Tx) bool {
	if len(tx.Inputs) != 2 {
		return false
	}

	if !IsP2WMCScript(tx.Inputs[0].ControlProgram()) || !IsP2WMCScript(tx.Inputs[1].ControlProgram()) {
		return false
	}

	if tx.Inputs[0].InputType() != types.SpendInputType || tx.Inputs[1].InputType() != types.SpendInputType {
		return false
	}

	if !isTradeClauseSelector(tx.Inputs[0]) {
		return false
	}

	return isTradeClauseSelector(tx.Inputs[1])
}

func isCancelClauseSelector(input *types.TxInput) bool {
	return len(input.Arguments()) >= 2 && hex.EncodeToString(input.Arguments()[1]) == hex.EncodeToString(vm.Int64Bytes(2))
}

func isTradeClauseSelector(input *types.TxInput) bool {
	if len(input.Arguments()) < 2 {
		return false
	}
	clauseSelector := hex.EncodeToString(input.Arguments()[1])
	return clauseSelector == hex.EncodeToString(vm.Int64Bytes(0)) || clauseSelector == hex.EncodeToString(vm.Int64Bytes(1))
}

// -------------------- mock -------------------

func IsP2WMCScript(prog []byte) bool {
	return false
}
