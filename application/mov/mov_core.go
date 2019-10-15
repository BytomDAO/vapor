package mov

import (
	"encoding/hex"

	"github.com/vapor/application/mov/common"
	"github.com/vapor/application/mov/database"
	"github.com/vapor/application/mov/match"
	"github.com/vapor/consensus/segwit"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm"
)

type MovCore struct {
	movStore *database.MovStore
}

// ChainStatus return the current block height and block hash in dex core
func (m *MovCore) ChainStatus() (uint64, *bc.Hash, error) {
	state, err := m.movStore.GetMovDatabaseState()
	if err != nil {
		return 0, nil, err
	}

	return state.Height, state.Hash, nil
}

func (m *MovCore) ValidateBlock(block *types.Block, attachBlocks, detachBlocks []*types.Block) error {
	if err := m.ValidateTxs(block, attachBlocks, detachBlocks); err != nil {
		return err
	}
	return nil
}

// ValidateTxs validate the matched transactions is generated according to the matching rule.
func (m *MovCore) ValidateTxs(block *types.Block, attachBlocks, detachBlocks []*types.Block) error {
	deltaOrderMap, err := m.getDeltaOrders(attachBlocks, detachBlocks)
	if err != nil {
		return err
	}

	if err := m.validateMatchedTxs(block.Transactions, deltaOrderMap); err != nil {
		return err
	}
	return nil
}

func (m *MovCore) validateMatchedTxs(txs []*types.Tx, deltaOrderMap map[string]*database.DeltaOrders) error {
	matchedTxMap := groupMatchedTx(txs)
	for _, packagedMatchedTxs := range matchedTxMap {
		tradePair := getTradePairFromMatchedTx(packagedMatchedTxs[0])
		realMatchedTxs, err := match.GenerateMatchedTxs(match.NewOrderTable(m.movStore, tradePair, deltaOrderMap))
		if err != nil {
			return err
		}

		for i := 0; i < len(packagedMatchedTxs); i++ {
			if i >= len(realMatchedTxs) || packagedMatchedTxs[i].ID != realMatchedTxs[i].ID {
				return errors.New("fail to validate match transaction")
			}
		}
	}
	return nil
}

func groupMatchedTx(txs []*types.Tx) map[string][]*types.Tx {
	matchedTxMap := make(map[string][]*types.Tx)
	for _, tx := range txs {
		if !isMatchedTx(tx) {
			continue
		}

		tradePair := getTradePairFromMatchedTx(tx)
		matchedTxMap[tradePair.String()] = append(matchedTxMap[tradePair.String()], tx)
	}
	return matchedTxMap
}

func getTradePairFromMatchedTx(tx *types.Tx) *common.TradePair {
	fromAssetID := tx.Inputs[0].AssetID()
	toAssetID := tx.Inputs[1].AssetID()
	return &common.TradePair{FromAssetID: &fromAssetID, ToAssetID: &toAssetID}
}

func (m *MovCore) getDeltaOrders(attachBlocks, detachBlocks []*types.Block) (map[string]*database.DeltaOrders, error) {
	var deleteOrders, addOrders []*common.Order
	for _, block := range detachBlocks {
		subDeleteOrders, subAddOrders, err := applyTransactions(block.Transactions)
		if err != nil {
			return nil, err
		}

		addOrders = append(addOrders, subAddOrders...)
		deleteOrders = append(deleteOrders, subDeleteOrders...)
	}

	for _, block := range attachBlocks {
		subAddOrders, subDeleteOrders, err := applyTransactions(block.Transactions)
		if err != nil {
			return nil, err
		}

		addOrders = append(addOrders, subAddOrders...)
		deleteOrders = append(deleteOrders, subDeleteOrders...)
	}

	return groupDeltaOrders(addOrders, deleteOrders), nil
}

func groupDeltaOrders(addOrders, deleteOrders []*common.Order) map[string]*database.DeltaOrders {
	deltaOrderMap := make(map[string]*database.DeltaOrders)

	for _, addOrder := range addOrders {
		tradePair := &common.TradePair{FromAssetID: addOrder.FromAssetID, ToAssetID: addOrder.ToAssetID}
		if _, ok := deltaOrderMap[tradePair.String()]; !ok {
			deltaOrderMap[tradePair.String()] = database.NewDeltaOrders()
		}
		deltaOrderMap[tradePair.String()].AppendAddOrder(addOrder)
	}

	for _, deleteOrder := range deleteOrders {
		tradePair := &common.TradePair{FromAssetID: deleteOrder.FromAssetID, ToAssetID: deleteOrder.ToAssetID}
		if _, ok := deltaOrderMap[tradePair.String()]; !ok {
			deltaOrderMap[tradePair.String()] = database.NewDeltaOrders()
		}
		deltaOrderMap[tradePair.String()].AppendDeleteOrder(deleteOrder)
	}
	return deltaOrderMap
}

// ApplyBlock parse pending order and cancel from the the transactions of block
// and add pending order to the dex db, remove cancel order from dex db.
func (m *MovCore) ApplyBlock(block *types.Block) error {
	addOrders, deleteOrders, err := applyTransactions(block.Transactions)
	if err != nil {
		return err
	}

	return m.movStore.ProcessOrders(addOrders, deleteOrders, &block.BlockHeader)
}

// DetachBlock parse pending order and cancel from the the transactions of block
// and add cancel order to the dex db, remove pending order from dex db.
func (m *MovCore) DetachBlock(block *types.Block) error {
	deleteOrders, addOrders, err := applyTransactions(block.Transactions)
	if err != nil {
		return err
	}

	var prevBlockHeader *types.BlockHeader

	return m.movStore.ProcessOrders(addOrders, deleteOrders, prevBlockHeader)
}

// BeforeProposalBlock get all pending orders from the dex db, parse pending orders and cancel orders from transactions
// Then merge the two, use match engine to generate matched transactions, finally return them.
func (m *MovCore) BeforeProposalBlock(txs []*types.Tx, numOfPackage int) ([]*types.Tx, error) {
	var packagedTxs []*types.Tx
	tradePairIterator := database.NewTradePairIterator(m.movStore)

	for tradePairIterator.HasNext() {
		addOrders, deleteOrders, err := applyTransactions(txs)
		if err != nil {
			return nil, err
		}

		orderTable := match.NewOrderTable(m.movStore, tradePairIterator.Next(), groupDeltaOrders(addOrders, deleteOrders))
		matchedTxs, err := match.GenerateMatchedTxs(orderTable)
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

func (m *MovCore) IsDust(tx *types.Tx) bool {
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
			addOrderMap[order.ID()] = order
		}

		deleteOrders, err := getDeleteOrdersFromTx(tx)
		if err != nil {
			return nil, nil, err
		}

		for _, order := range deleteOrders {
			deleteOrderMap[order.ID()] = order
		}
	}

	addOrders, deleteOrders := mergeOrders(addOrderMap, deleteOrderMap)
	return addOrders, deleteOrders, nil
}

func mergeOrders(addOrderMap, deleteOrderMap map[string]*common.Order) ([]*common.Order, []*common.Order) {
	var deleteOrders, addOrders []*common.Order
	for orderID, order := range addOrderMap {
		if deleteOrderMap[orderID] != nil {
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
		if output.OutputType() == types.IntraChainOutputType && segwit.IsP2WMCScript(output.ControlProgram()) {
			order, err := common.NewOrderFromOutput(tx, i)
			if err != nil {
				return nil, err
			}

			orders = append(orders, order)
		}
	}
	return orders, nil
}

func getDeleteOrdersFromTx(tx *types.Tx) ([]*common.Order, error) {
	var orders []*common.Order
	for _, input := range tx.Inputs {
		if input.InputType() != types.SpendInputType || !segwit.IsP2WMCScript(input.ControlProgram()) {
			continue
		}

		if isCancelClauseSelector(input) || isTradeClauseSelector(input) {
			order, err := common.NewOrderFromInput(input)
			if err != nil {
				return nil, err
			}

			orders = append(orders, order)
		}
	}
	return orders, nil
}

func isMatchedTx(tx *types.Tx) bool {
	p2wmCount := 0
	for _, input := range tx.Inputs {
		if segwit.IsP2WMCScript(input.ControlProgram()) && isTradeClauseSelector(input) && input.InputType() == types.SpendInputType {
			p2wmCount++
		}
	}
	return p2wmCount >= 2
}

func isCancelClauseSelector(input *types.TxInput) bool {
	return len(input.Arguments()) == 3 && hex.EncodeToString(input.Arguments()[2]) == hex.EncodeToString(vm.Int64Bytes(2))
}

func isTradeClauseSelector(input *types.TxInput) bool {
	arguments := input.Arguments()
	clauseSelector := hex.EncodeToString(arguments[len(arguments)-1])
	return clauseSelector == hex.EncodeToString(vm.Int64Bytes(0)) || clauseSelector == hex.EncodeToString(vm.Int64Bytes(1))
}

