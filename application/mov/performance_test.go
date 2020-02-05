package mov

import (
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/application/mov/database"
	"github.com/bytom/vapor/application/mov/mock"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/vm"
)

func TestPlaceOrderPerformance(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	store := database.NewLevelDBMovStore(testDB)
	movCore := &MovCore{movStore: store}
	initBlock := &types.Block{}
	if err := movCore.ApplyBlock(initBlock); err != nil {
		t.Fatal(err)
	}

	block := &types.Block{BlockHeader: types.BlockHeader{Height: 1, PreviousBlockHash: initBlock.Hash()}}
	for i := 0; i < 100000; i++ {
		tx := mock.Btc2EthMakerTxs[0]
		tx.Inputs[0].TypedInput.(*types.SpendInput).SourcePosition = uint64(i)
		tx = types.NewTx(tx.TxData)
		block.Transactions = append(block.Transactions, tx)
	}

	startTime := time.Now().UnixNano()
	if err := movCore.ApplyBlock(block); err != nil {
		t.Fatal(err)
	}
	endTime := time.Now().UnixNano()
	t.Logf("cost %dms\n", (endTime - startTime) / 1E6)

	testDB.Close()
	os.RemoveAll("temp")
}

func TestCancelOrderPerformance(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	store := database.NewLevelDBMovStore(testDB)
	movCore := &MovCore{movStore: store}
	initBlock := &types.Block{}
	if err := movCore.ApplyBlock(initBlock); err != nil {
		t.Fatal(err)
	}

	var orders []*common.Order
	block := &types.Block{BlockHeader: types.BlockHeader{Height: 1, PreviousBlockHash: initBlock.Hash()}}
	for i := 0; i < 10000; i++ {
		tx := mock.Btc2EthMakerTxs[0]
		tx.Inputs[0].TypedInput.(*types.SpendInput).SourcePosition = uint64(i)
		tx = types.NewTx(tx.TxData)
		block.Transactions = append(block.Transactions, tx)

		order, _ := common.NewOrderFromOutput(tx, 0)
		orders = append(orders, order)
	}
	if err := movCore.ApplyBlock(block); err != nil {
		t.Fatal(err)
	}

	fmt.Println("---------------------------------------")

	cancelBlock := &types.Block{BlockHeader: 	types.BlockHeader{Height: 2, PreviousBlockHash: block.Hash()}}
	for i := 0; i < len(orders); i++ {
		cancelTx := createCancelTx(orders[i])
		cancelBlock.Transactions = append(cancelBlock.Transactions, cancelTx)
	}

	startTime := time.Now().UnixNano()
	if err := movCore.ApplyBlock(cancelBlock); err != nil {
		t.Fatal(err)
	}
	endTime := time.Now().UnixNano()
	t.Logf("cancel order cost %dms\n", (endTime - startTime) / 1E6)

	testDB.Close()
	os.RemoveAll("temp")
}

func TestMatchTx(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	store := database.NewLevelDBMovStore(testDB)
	movCore := &MovCore{movStore: store}
	initBlock := &types.Block{}
	if err := movCore.ApplyBlock(initBlock); err != nil {
		t.Fatal(err)
	}

	block := &types.Block{BlockHeader: types.BlockHeader{Height: 1, PreviousBlockHash: initBlock.Hash()}}
	for i := 0; i < 10000; i++ {
		tx := mock.Btc2EthMakerTxs[0]
		tx.Inputs[0].TypedInput.(*types.SpendInput).SourcePosition = uint64(i)
		tx = types.NewTx(tx.TxData)
		block.Transactions = append(block.Transactions, tx)

		tx = mock.Etc2EosMakerTxs[0]
		tx.Inputs[0].TypedInput.(*types.SpendInput).SourcePosition = uint64(i)
		tx = types.NewTx(tx.TxData)
		block.Transactions = append(block.Transactions, tx)
	}

	for i := 0; i < 10000; i++ {
		tx := mock.Eth2BtcMakerTxs[1]
		tx.Inputs[0].TypedInput.(*types.SpendInput).SourcePosition = uint64(i)
		tx = types.NewTx(tx.TxData)
		block.Transactions = append(block.Transactions, tx)

		tx = mock.Eos2EtcMakerTxs[0]
		tx.Inputs[0].TypedInput.(*types.SpendInput).SourcePosition = uint64(i)
		tx = types.NewTx(tx.TxData)
		block.Transactions = append(block.Transactions, tx)
	}

	if err := movCore.ApplyBlock(block); err != nil {
		t.Fatal(err)
	}

	fmt.Println("------------------------------")

	startTime := time.Now().UnixNano()
	matchedTxs, err := movCore.BeforeProposalBlock(nil, []byte{0x51}, 2, math.MaxInt64, func() bool {return false})
	if err != nil {
		t.Fatal(err)
	}
	endTime := time.Now().UnixNano()
	t.Logf("matched tx cost %dms\n", (endTime - startTime) / 1E6)

	t.Log(len(matchedTxs))

	testDB.Close()
	os.RemoveAll("temp")
}

func createCancelTx(order *common.Order) *types.Tx {
	tx := &types.TxData{Version: 1}
	tx.Inputs = append(tx.Inputs, types.NewSpendInput([][]byte{{}, {}, vm.Int64Bytes(2)}, *order.Utxo.SourceID, *order.FromAssetID, order.Utxo.Amount, order.Utxo.SourcePos, order.Utxo.ControlProgram))
	tx.Outputs = append(tx.Outputs, types.NewIntraChainOutput(*order.FromAssetID, order.Utxo.Amount, []byte{0x51}))
	byteData, err := tx.MarshalText()
	if err != nil {
		panic(err)
	}

	tx.SerializedSize = uint64(len(byteData))
	return types.NewTx(*tx)
}
