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

func (m *MovCore) ValidateBlock(block *types.Block) error {
	if err := m.ValidateTxs(block.Transactions); err != nil {
		return err
	}
	return nil
}

// ValidateTxs validate the matched transactions is generated according to the matching rule.
func (m *MovCore) ValidateTxs(txs []*types.Tx) error {
	return nil
}

func (m *MovCore) validateMatchedTxs(txs []*types.Tx) error {
	matchedTxMap := groupMatchedTx(txs)
	for _, packagedMatchedTxs := range matchedTxMap {
		tradePair := getTradePairFromMatchedTx(packagedMatchedTxs[0])
		realMatchedTxs, err := match.GenerateMatchedTxs(match.NewOrderTable(m.movStore, tradePair))
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
	if fromAssetID.String() > toAssetID.String() {
		return &common.TradePair{FromAssetID: &fromAssetID, ToAssetID: &toAssetID}
	}
	return &common.TradePair{FromAssetID: &toAssetID, ToAssetID: &fromAssetID}
}

// ApplyBlock parse pending order and cancel from the the transactions of block
// and add pending order to the dex db, remove cancel order from dex db.
func (m *MovCore) ApplyBlock(block *types.Block) error {
	if err := m.validateMatchedTxs(block.Transactions); err != nil {
		return err
	}

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
		orderTable := match.NewOrderTable(m.movStore, tradePairIterator.Next())
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

