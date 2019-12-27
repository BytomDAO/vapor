package mov

import (
	"math"
	"os"
	"testing"

	"github.com/vapor/application/mov/common"
	"github.com/vapor/application/mov/database"
	"github.com/vapor/application/mov/mock"
	"github.com/vapor/consensus"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm"
	"github.com/vapor/testutil"
)

func TestApplyBlock(t *testing.T) {
	initBlockHeader := &types.BlockHeader{Height: 1, PreviousBlockHash: bc.Hash{}}
	cases := []struct {
		desc        string
		block       *types.Block
		blockFunc   testFun
		initOrders  []*common.Order
		wantOrders  []*common.Order
		wantDBState *common.MovDatabaseState
		wantError   error
	}{
		{
			desc: "apply block has pending order transaction",
			block: &types.Block{
				BlockHeader: types.BlockHeader{Height: 2, PreviousBlockHash: initBlockHeader.Hash()},
				Transactions: []*types.Tx{
					mock.Btc2EthMakerTxs[0], mock.Eth2BtcMakerTxs[0],
				},
			},
			blockFunc:   applyBlock,
			wantOrders:  []*common.Order{mock.MustNewOrderFromOutput(mock.Btc2EthMakerTxs[0], 0), mock.MustNewOrderFromOutput(mock.Eth2BtcMakerTxs[0], 0)},
			wantDBState: &common.MovDatabaseState{Height: 2, Hash: hashPtr(testutil.MustDecodeHash("88dbcde57bb2b53b107d7494f20f1f1a892307a019705980c3510890449c0020"))},
		},
		{
			desc: "apply block has full matched transaction",
			block: &types.Block{
				BlockHeader: types.BlockHeader{Height: 2, PreviousBlockHash: initBlockHeader.Hash()},
				Transactions: []*types.Tx{
					mock.MatchedTxs[1],
				},
			},
			blockFunc:   applyBlock,
			initOrders:  []*common.Order{mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Eth2BtcOrders[0]},
			wantOrders:  []*common.Order{mock.Btc2EthOrders[1]},
			wantDBState: &common.MovDatabaseState{Height: 2, Hash: hashPtr(testutil.MustDecodeHash("88dbcde57bb2b53b107d7494f20f1f1a892307a019705980c3510890449c0020"))},
		},
		{
			desc: "apply block has partial matched transaction",
			block: &types.Block{
				BlockHeader: types.BlockHeader{Height: 2, PreviousBlockHash: initBlockHeader.Hash()},
				Transactions: []*types.Tx{
					mock.MatchedTxs[0],
				},
			},
			blockFunc:   applyBlock,
			initOrders:  []*common.Order{mock.Btc2EthOrders[0], mock.Eth2BtcOrders[1]},
			wantOrders:  []*common.Order{mock.MustNewOrderFromOutput(mock.MatchedTxs[0], 1)},
			wantDBState: &common.MovDatabaseState{Height: 2, Hash: hashPtr(testutil.MustDecodeHash("88dbcde57bb2b53b107d7494f20f1f1a892307a019705980c3510890449c0020"))},
		},
		{
			desc: "apply block has two partial matched transaction",
			block: &types.Block{
				BlockHeader: types.BlockHeader{Height: 2, PreviousBlockHash: initBlockHeader.Hash()},
				Transactions: []*types.Tx{
					mock.MatchedTxs[2], mock.MatchedTxs[3],
				},
			},
			blockFunc:   applyBlock,
			initOrders:  []*common.Order{mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Eth2BtcOrders[2]},
			wantOrders:  []*common.Order{mock.MustNewOrderFromOutput(mock.MatchedTxs[3], 1)},
			wantDBState: &common.MovDatabaseState{Height: 2, Hash: hashPtr(testutil.MustDecodeHash("88dbcde57bb2b53b107d7494f20f1f1a892307a019705980c3510890449c0020"))},
		},
		{
			desc: "apply block has partial matched transaction by pending orders from tx pool",
			block: &types.Block{
				BlockHeader: types.BlockHeader{Height: 2, PreviousBlockHash: initBlockHeader.Hash()},
				Transactions: []*types.Tx{
					mock.Btc2EthMakerTxs[0],
					mock.Eth2BtcMakerTxs[1],
					mock.MatchedTxs[4],
				},
			},
			blockFunc:   applyBlock,
			initOrders:  []*common.Order{},
			wantOrders:  []*common.Order{mock.MustNewOrderFromOutput(mock.MatchedTxs[4], 1)},
			wantDBState: &common.MovDatabaseState{Height: 2, Hash: hashPtr(testutil.MustDecodeHash("88dbcde57bb2b53b107d7494f20f1f1a892307a019705980c3510890449c0020"))},
		},
		{
			desc: "detach block has pending order transaction",
			block: &types.Block{
				BlockHeader: *initBlockHeader,
				Transactions: []*types.Tx{
					mock.Btc2EthMakerTxs[0], mock.Eth2BtcMakerTxs[1],
				},
			},
			blockFunc:   detachBlock,
			initOrders:  []*common.Order{mock.MustNewOrderFromOutput(mock.Btc2EthMakerTxs[0], 0), mock.MustNewOrderFromOutput(mock.Eth2BtcMakerTxs[1], 0)},
			wantOrders:  []*common.Order{},
			wantDBState: &common.MovDatabaseState{Height: 0, Hash: &bc.Hash{}},
		},
		{
			desc: "detach block has full matched transaction",
			block: &types.Block{
				BlockHeader: *initBlockHeader,
				Transactions: []*types.Tx{
					mock.MatchedTxs[1],
				},
			},
			blockFunc:   detachBlock,
			initOrders:  []*common.Order{mock.Btc2EthOrders[1]},
			wantOrders:  []*common.Order{mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Eth2BtcOrders[0]},
			wantDBState: &common.MovDatabaseState{Height: 0, Hash: &bc.Hash{}},
		},
		{
			desc: "detach block has partial matched transaction",
			block: &types.Block{
				BlockHeader: *initBlockHeader,
				Transactions: []*types.Tx{
					mock.MatchedTxs[0],
				},
			},
			blockFunc:   detachBlock,
			initOrders:  []*common.Order{mock.MustNewOrderFromOutput(mock.MatchedTxs[0], 1)},
			wantOrders:  []*common.Order{mock.Btc2EthOrders[0], mock.Eth2BtcOrders[1]},
			wantDBState: &common.MovDatabaseState{Height: 0, Hash: &bc.Hash{}},
		},
		{
			desc: "detach block has two partial matched transaction",
			block: &types.Block{
				BlockHeader: *initBlockHeader,
				Transactions: []*types.Tx{
					mock.MatchedTxs[2], mock.MatchedTxs[3],
				},
			},
			blockFunc:   detachBlock,
			initOrders:  []*common.Order{mock.MustNewOrderFromOutput(mock.MatchedTxs[3], 1)},
			wantOrders:  []*common.Order{mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Eth2BtcOrders[2]},
			wantDBState: &common.MovDatabaseState{Height: 0, Hash: &bc.Hash{}},
		},
	}

	defer os.RemoveAll("temp")
	for i, c := range cases {
		testDB := dbm.NewDB("testdb", "leveldb", "temp")
		store := database.NewLevelDBMovStore(testDB)
		if err := store.InitDBState(0, &bc.Hash{}); err != nil {
			t.Fatal(err)
		}

		if err := store.ProcessOrders(c.initOrders, nil, initBlockHeader); err != nil {
			t.Fatal(err)
		}

		movCore := &MovCore{movStore: store}
		if err := c.blockFunc(movCore, c.block); err != c.wantError {
			t.Errorf("#%d(%s):apply block want error(%v), got error(%v)", i, c.desc, c.wantError, err)
		}

		gotOrders := queryAllOrders(store)
		if !ordersEquals(c.wantOrders, gotOrders) {
			t.Errorf("#%d(%s):apply block want orders(%v), got orders(%v)", i, c.desc, c.wantOrders, gotOrders)
		}

		dbState, err := store.GetMovDatabaseState()
		if err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(c.wantDBState, dbState) {
			t.Errorf("#%d(%s):apply block want db state(%v), got db state(%v)", i, c.desc, c.wantDBState, dbState)
		}

		testDB.Close()
		os.RemoveAll("temp")
	}
}

func TestValidateBlock(t *testing.T) {
	cases := []struct {
		desc          string
		block         *types.Block
		verifyResults []*bc.TxVerifyResult
		wantError     error
	}{
		{
			desc: "block only has maker tx",
			block: &types.Block{
				Transactions: []*types.Tx{
					mock.Eth2BtcMakerTxs[0],
					mock.Btc2EthMakerTxs[0],
				},
			},
			verifyResults: []*bc.TxVerifyResult{{StatusFail: false}, {StatusFail: false}},
			wantError:     nil,
		},
		{
			desc: "block only has matched tx",
			block: &types.Block{
				Transactions: []*types.Tx{
					mock.MatchedTxs[0],
					mock.MatchedTxs[1],
					mock.MatchedTxs[2],
				},
			},
			verifyResults: []*bc.TxVerifyResult{{StatusFail: false}, {StatusFail: false}, {StatusFail: false}},
			wantError:     nil,
		},
		{
			desc: "block has maker tx and matched tx",
			block: &types.Block{
				Transactions: []*types.Tx{
					mock.Eth2BtcMakerTxs[0],
					mock.Btc2EthMakerTxs[0],
					mock.MatchedTxs[0],
					mock.MatchedTxs[1],
					mock.MatchedTxs[2],
				},
			},
			verifyResults: []*bc.TxVerifyResult{{StatusFail: false}, {StatusFail: false}, {StatusFail: false}, {StatusFail: false}, {StatusFail: false}},
			wantError:     nil,
		},
		{
			desc: "status fail of maker tx is true",
			block: &types.Block{
				Transactions: []*types.Tx{
					mock.Eth2BtcMakerTxs[0],
					mock.Btc2EthMakerTxs[0],
				},
			},
			verifyResults: []*bc.TxVerifyResult{{StatusFail: false}, {StatusFail: true}},
			wantError:     errStatusFailMustFalse,
		},
		{
			desc: "status fail of matched tx is true",
			block: &types.Block{
				Transactions: []*types.Tx{
					mock.MatchedTxs[1],
					mock.MatchedTxs[2],
				},
			},
			verifyResults: []*bc.TxVerifyResult{{StatusFail: false}, {StatusFail: true}},
			wantError:     errStatusFailMustFalse,
		},
		{
			desc: "asset id in matched tx is not unique",
			block: &types.Block{
				Transactions: []*types.Tx{
					types.NewTx(types.TxData{
						Inputs: []*types.TxInput{
							types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *mock.Btc2EthOrders[0].Utxo.SourceID, *mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.Btc2EthOrders[0].Utxo.SourcePos, mock.Btc2EthOrders[0].Utxo.ControlProgram),
							types.NewSpendInput([][]byte{vm.Int64Bytes(1), vm.Int64Bytes(1)}, *mock.Eth2BtcOrders[0].Utxo.SourceID, *mock.Btc2EthOrders[0].FromAssetID, mock.Eth2BtcOrders[0].Utxo.Amount, mock.Eth2BtcOrders[0].Utxo.SourcePos, mock.Eth2BtcOrders[0].Utxo.ControlProgram),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(*mock.Btc2EthOrders[0].ToAssetID, 500, testutil.MustDecodeHexString("51")),
							types.NewIntraChainOutput(*mock.Eth2BtcOrders[0].ToAssetID, 10, testutil.MustDecodeHexString("53")),
							types.NewIntraChainOutput(*mock.Btc2EthOrders[0].ToAssetID, 10, []byte{0x51}),
						},
					}),
				},
			},
			verifyResults: []*bc.TxVerifyResult{{StatusFail: false}, {StatusFail: true}},
			wantError:     errAssetIDMustUniqueInMatchedTx,
		},
		{
			desc: "common input in the matched tx",
			block: &types.Block{
				Transactions: []*types.Tx{
					types.NewTx(types.TxData{
						Inputs: []*types.TxInput{
							types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *mock.Btc2EthOrders[0].Utxo.SourceID, *mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.Btc2EthOrders[0].Utxo.SourcePos, mock.Btc2EthOrders[0].Utxo.ControlProgram),
							types.NewSpendInput([][]byte{vm.Int64Bytes(1), vm.Int64Bytes(1)}, *mock.Eth2BtcOrders[0].Utxo.SourceID, *mock.Eth2BtcOrders[0].FromAssetID, mock.Eth2BtcOrders[0].Utxo.Amount, mock.Eth2BtcOrders[0].Utxo.SourcePos, mock.Eth2BtcOrders[0].Utxo.ControlProgram),
							types.NewSpendInput(nil, testutil.MustDecodeHash("28b7b53d8dc90006bf97e0a4eaae2a72ec3d869873188698b694beaf20789f21"), *consensus.BTMAssetID, 100, 0, []byte{0x51}),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(*mock.Btc2EthOrders[0].ToAssetID, 500, testutil.MustDecodeHexString("51")),
							types.NewIntraChainOutput(*mock.Eth2BtcOrders[0].ToAssetID, 10, testutil.MustDecodeHexString("53")),
							types.NewIntraChainOutput(*mock.Btc2EthOrders[0].ToAssetID, 10, []byte{0x51}),
							types.NewIntraChainOutput(*consensus.BTMAssetID, 100, []byte{0x51}),
						},
					}),
				},
			},
			verifyResults: []*bc.TxVerifyResult{{StatusFail: false}},
			wantError:     errInputProgramMustP2WMCScript,
		},
		{
			desc: "cancel order in the matched tx",
			block: &types.Block{
				Transactions: []*types.Tx{
					types.NewTx(types.TxData{
						Inputs: []*types.TxInput{
							types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *mock.Btc2EthOrders[0].Utxo.SourceID, *mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.Btc2EthOrders[0].Utxo.SourcePos, mock.Btc2EthOrders[0].Utxo.ControlProgram),
							types.NewSpendInput([][]byte{vm.Int64Bytes(1), vm.Int64Bytes(1)}, *mock.Eth2BtcOrders[0].Utxo.SourceID, *mock.Eth2BtcOrders[0].FromAssetID, mock.Eth2BtcOrders[0].Utxo.Amount, mock.Eth2BtcOrders[0].Utxo.SourcePos, mock.Eth2BtcOrders[0].Utxo.ControlProgram),
							types.NewSpendInput([][]byte{{}, {}, vm.Int64Bytes(2)}, *mock.Btc2EthOrders[0].Utxo.SourceID, *mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.Btc2EthOrders[0].Utxo.SourcePos, mock.Btc2EthOrders[0].Utxo.ControlProgram),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(*mock.Btc2EthOrders[0].ToAssetID, 500, testutil.MustDecodeHexString("51")),
							types.NewIntraChainOutput(*mock.Eth2BtcOrders[0].ToAssetID, 10, testutil.MustDecodeHexString("53")),
							types.NewIntraChainOutput(*mock.Btc2EthOrders[0].ToAssetID, 10, []byte{0x51}),
							types.NewIntraChainOutput(*consensus.BTMAssetID, 100, []byte{0x51}),
						},
					}),
				},
			},
			verifyResults: []*bc.TxVerifyResult{{StatusFail: false}},
			wantError:     errExistCancelOrderInMatchedTx,
		},
		{
			desc: "common input in the cancel order tx",
			block: &types.Block{
				Transactions: []*types.Tx{
					types.NewTx(types.TxData{
						Inputs: []*types.TxInput{
							types.NewSpendInput([][]byte{{}, {}, vm.Int64Bytes(2)}, *mock.Btc2EthOrders[0].Utxo.SourceID, *mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.Btc2EthOrders[0].Utxo.SourcePos, mock.Btc2EthOrders[0].Utxo.ControlProgram),
							types.NewSpendInput(nil, testutil.MustDecodeHash("28b7b53d8dc90006bf97e0a4eaae2a72ec3d869873188698b694beaf20789f21"), *consensus.BTMAssetID, 100, 0, []byte{0x51}),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(*mock.Btc2EthOrders[0].FromAssetID, 10, testutil.MustDecodeHexString("51")),
							types.NewIntraChainOutput(*consensus.BTMAssetID, 100, []byte{0x51}),
						},
					}),
				},
			},
			verifyResults: []*bc.TxVerifyResult{{StatusFail: false}},
			wantError:     errInputProgramMustP2WMCScript,
		},
		{
			desc: "amount of fee greater than max fee amount",
			block: &types.Block{
				Transactions: []*types.Tx{
					types.NewTx(types.TxData{
						Inputs: []*types.TxInput{
							types.NewSpendInput([][]byte{vm.Int64Bytes(0), vm.Int64Bytes(1)}, *mock.Btc2EthOrders[0].Utxo.SourceID, *mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.Btc2EthOrders[0].Utxo.SourcePos, mock.Btc2EthOrders[0].Utxo.ControlProgram),
							types.NewSpendInput([][]byte{vm.Int64Bytes(10), vm.Int64Bytes(1), vm.Int64Bytes(0)}, *mock.Eth2BtcOrders[2].Utxo.SourceID, *mock.Eth2BtcOrders[2].FromAssetID, mock.Eth2BtcOrders[2].Utxo.Amount, mock.Eth2BtcOrders[2].Utxo.SourcePos, mock.Eth2BtcOrders[2].Utxo.ControlProgram),
						},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(*mock.Btc2EthOrders[0].ToAssetID, 500, testutil.MustDecodeHexString("51")),
							types.NewIntraChainOutput(*mock.Eth2BtcOrders[2].ToAssetID, 10, testutil.MustDecodeHexString("55")),
							// re-order
							types.NewIntraChainOutput(*mock.Eth2BtcOrders[2].FromAssetID, 270, mock.Eth2BtcOrders[2].Utxo.ControlProgram),
							// fee
							types.NewIntraChainOutput(*mock.Eth2BtcOrders[2].FromAssetID, 40, []byte{0x59}),
						},
					}),
				},
			},
			verifyResults: []*bc.TxVerifyResult{{StatusFail: false}},
			wantError:     errAmountOfFeeGreaterThanMaximum,
		},
		{
			desc: "ratio numerator is zero",
			block: &types.Block{
				Transactions: []*types.Tx{
					types.NewTx(types.TxData{
						Inputs:  []*types.TxInput{types.NewSpendInput(nil, *mock.Btc2EthOrders[0].Utxo.SourceID, *mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.Btc2EthOrders[0].Utxo.SourcePos, []byte{0x51})},
						Outputs: []*types.TxOutput{types.NewIntraChainOutput(*mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.MustCreateP2WMCProgram(mock.ETH, testutil.MustDecodeHexString("51"), 0, 1))},
					}),
				},
			},
			verifyResults: []*bc.TxVerifyResult{{StatusFail: false}},
			wantError:     errRatioOfTradeLessThanZero,
		},
		{
			desc: "ratio denominator is zero",
			block: &types.Block{
				Transactions: []*types.Tx{
					types.NewTx(types.TxData{
						Inputs:  []*types.TxInput{types.NewSpendInput(nil, *mock.Btc2EthOrders[0].Utxo.SourceID, *mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.Btc2EthOrders[0].Utxo.SourcePos, []byte{0x51})},
						Outputs: []*types.TxOutput{types.NewIntraChainOutput(*mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.MustCreateP2WMCProgram(mock.ETH, testutil.MustDecodeHexString("51"), 1, 0))},
					}),
				},
			},
			verifyResults: []*bc.TxVerifyResult{{StatusFail: false}},
			wantError:     errRatioOfTradeLessThanZero,
		},
		{
			desc: "ratio numerator product input amount is overflow",
			block: &types.Block{
				Transactions: []*types.Tx{
					types.NewTx(types.TxData{
						Inputs:  []*types.TxInput{types.NewSpendInput(nil, *mock.Btc2EthOrders[0].Utxo.SourceID, *mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.Btc2EthOrders[0].Utxo.SourcePos, []byte{0x51})},
						Outputs: []*types.TxOutput{types.NewIntraChainOutput(*mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.MustCreateP2WMCProgram(mock.ETH, testutil.MustDecodeHexString("51"), math.MaxInt64, 10))},
					}),
				},
			},
			verifyResults: []*bc.TxVerifyResult{{StatusFail: false}},
			wantError:     errNumeratorOfRatioIsOverflow,
		},
	}

	for i, c := range cases {
		movCore := &MovCore{}
		if err := movCore.ValidateBlock(c.block, c.verifyResults); err != c.wantError {
			t.Errorf("#%d(%s):validate block want error(%v), got error(%v)", i, c.desc, c.wantError, err)
		}
	}
}

type testFun func(movCore *MovCore, block *types.Block) error

func applyBlock(movCore *MovCore, block *types.Block) error {
	return movCore.ApplyBlock(block)
}

func detachBlock(movCore *MovCore, block *types.Block) error {
	return movCore.DetachBlock(block)
}

func queryAllOrders(store *database.LevelDBMovStore) []*common.Order {
	var orders []*common.Order
	tradePairIterator := database.NewTradePairIterator(store)
	for tradePairIterator.HasNext() {
		orderIterator := database.NewOrderIterator(store, tradePairIterator.Next())
		for orderIterator.HasNext() {
			orders = append(orders, orderIterator.NextBatch()...)
		}
	}
	return orders
}

func ordersEquals(orders1 []*common.Order, orders2 []*common.Order) bool {
	orderMap1 := make(map[string]*common.Order)
	for _, order := range orders1 {
		orderMap1[order.Key()] = order
	}

	orderMap2 := make(map[string]*common.Order)
	for _, order := range orders2 {
		orderMap2[order.Key()] = order
	}
	return testutil.DeepEqual(orderMap1, orderMap2)
}

func hashPtr(hash bc.Hash) *bc.Hash {
	return &hash
}
