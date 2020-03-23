package mov

import (
	"encoding/hex"
	"math"
	"os"
	"testing"

	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/application/mov/database"
	"github.com/bytom/vapor/application/mov/match"
	"github.com/bytom/vapor/application/mov/mock"
	"github.com/bytom/vapor/consensus"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/vm"
	"github.com/bytom/vapor/testutil"
)

var initBlockHeader = &types.BlockHeader{Height: 1, PreviousBlockHash: bc.Hash{}}

func TestApplyBlock(t *testing.T) {
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
			desc: "apply block has two different trade pairs & different trade pair won't affect each order",
			block: &types.Block{
				BlockHeader: types.BlockHeader{Height: 2, PreviousBlockHash: initBlockHeader.Hash()},
				Transactions: []*types.Tx{
					mock.Btc2EthMakerTxs[0],
					mock.Eth2BtcMakerTxs[0],
					mock.Eos2EtcMakerTxs[0],
					mock.Eth2EosMakerTxs[0],
				},
			},
			blockFunc: applyBlock,
			wantOrders: []*common.Order{
				mock.MustNewOrderFromOutput(mock.Btc2EthMakerTxs[0], 0),
				mock.MustNewOrderFromOutput(mock.Eth2BtcMakerTxs[0], 0),
				mock.MustNewOrderFromOutput(mock.Eos2EtcMakerTxs[0], 0),
				mock.MustNewOrderFromOutput(mock.Eth2EosMakerTxs[0], 0),
			},
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
			desc: "apply block which node packed maker tx and match transaction in random orde",
			block: &types.Block{
				BlockHeader: types.BlockHeader{Height: 2, PreviousBlockHash: initBlockHeader.Hash()},
				Transactions: []*types.Tx{
					mock.Eos2EtcMakerTxs[0],
					mock.Btc2EthMakerTxs[0],
					mock.Eth2BtcMakerTxs[1],
					mock.MatchedTxs[4],
					mock.Eth2EosMakerTxs[0],
					mock.Etc2EosMakerTxs[0],
					mock.MatchedTxs[5],
				},
			},
			blockFunc:  applyBlock,
			initOrders: []*common.Order{},
			wantOrders: []*common.Order{
				mock.MustNewOrderFromOutput(mock.MatchedTxs[4], 1),
				mock.MustNewOrderFromOutput(mock.Eth2EosMakerTxs[0], 0),
			},
			wantDBState: &common.MovDatabaseState{Height: 2, Hash: hashPtr(testutil.MustDecodeHash("88dbcde57bb2b53b107d7494f20f1f1a892307a019705980c3510890449c0020"))},
		},
		{
			desc: "apply block has partial matched transaction chain",
			block: &types.Block{
				BlockHeader: types.BlockHeader{Height: 2, PreviousBlockHash: initBlockHeader.Hash()},
				Transactions: []*types.Tx{
					mock.Btc2EthMakerTxs[0],
					mock.Eth2BtcMakerTxs[1],
					mock.MatchedTxs[4],
					mock.Eth2BtcMakerTxs[0],
					mock.MatchedTxs[7],
				},
			},
			blockFunc:   applyBlock,
			initOrders:  []*common.Order{},
			wantOrders:  []*common.Order{mock.MustNewOrderFromOutput(mock.MatchedTxs[7], 2)},
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
			desc: "detach block has two different trade pairs & different trade pair won't affect each order",
			block: &types.Block{
				BlockHeader: *initBlockHeader,
				Transactions: []*types.Tx{
					mock.Btc2EthMakerTxs[0],
					mock.Eth2BtcMakerTxs[0],
					mock.Eos2EtcMakerTxs[0],
					mock.Eth2EosMakerTxs[0],
				},
			},
			blockFunc: detachBlock,
			initOrders: []*common.Order{
				mock.MustNewOrderFromOutput(mock.Btc2EthMakerTxs[0], 0),
				mock.MustNewOrderFromOutput(mock.Eth2BtcMakerTxs[0], 0),
				mock.MustNewOrderFromOutput(mock.Eos2EtcMakerTxs[0], 0),
				mock.MustNewOrderFromOutput(mock.Eth2EosMakerTxs[0], 0),
			},
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
		{
			desc: "detach block which node packed maker tx and match transaction in random orde",
			block: &types.Block{
				BlockHeader: *initBlockHeader,
				Transactions: []*types.Tx{
					mock.Eos2EtcMakerTxs[0],
					mock.Btc2EthMakerTxs[0],
					mock.MatchedTxs[4],
					mock.Eth2EosMakerTxs[0],
					mock.Eth2BtcMakerTxs[1],
					mock.MatchedTxs[5],
					mock.Etc2EosMakerTxs[0],
				},
			},
			blockFunc: detachBlock,
			initOrders: []*common.Order{
				mock.MustNewOrderFromOutput(mock.MatchedTxs[4], 1),
				mock.MustNewOrderFromOutput(mock.Eth2EosMakerTxs[0], 0),
			},
			wantOrders:  []*common.Order{},
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

		movCore := &Core{movStore: store}
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
	consensus.ActiveNetParams.MovRewardPrograms = []consensus.MovRewardProgram{
		{
			BeginBlock: 0,
			EndBlock:   100,
			Program:    hex.EncodeToString(mock.RewardProgram),
		},
	}

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
							types.NewIntraChainOutput(*mock.Btc2EthOrders[0].ToAssetID, 499, testutil.MustDecodeHexString("51")),
							types.NewIntraChainOutput(*mock.Eth2BtcOrders[0].ToAssetID, 9, testutil.MustDecodeHexString("53")),
							// fee
							types.NewIntraChainOutput(*mock.Btc2EthOrders[0].ToAssetID, 11, mock.RewardProgram),
							types.NewIntraChainOutput(*mock.Eth2BtcOrders[0].ToAssetID, 1, mock.RewardProgram),
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
							types.NewIntraChainOutput(*mock.Btc2EthOrders[0].ToAssetID, 499, testutil.MustDecodeHexString("51")),
							types.NewIntraChainOutput(*mock.Eth2BtcOrders[0].ToAssetID, 9, testutil.MustDecodeHexString("53")),
							types.NewIntraChainOutput(*mock.Btc2EthOrders[0].ToAssetID, 11, mock.RewardProgram),
							types.NewIntraChainOutput(*mock.Eth2BtcOrders[0].ToAssetID, 1, mock.RewardProgram),
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
							types.NewIntraChainOutput(*mock.Btc2EthOrders[0].ToAssetID, 499, testutil.MustDecodeHexString("51")),
							types.NewIntraChainOutput(*mock.Eth2BtcOrders[0].ToAssetID, 9, testutil.MustDecodeHexString("53")),
							types.NewIntraChainOutput(*mock.Btc2EthOrders[0].ToAssetID, 11, mock.RewardProgram),
							types.NewIntraChainOutput(*mock.Eth2BtcOrders[0].ToAssetID, 1, mock.RewardProgram),
							types.NewIntraChainOutput(*mock.Btc2EthOrders[0].FromAssetID, 10, testutil.MustDecodeHexString("51")),
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
							types.NewIntraChainOutput(*consensus.BTMAssetID, 100, mock.RewardProgram),
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
							types.NewIntraChainOutput(*mock.Btc2EthOrders[0].ToAssetID, 499, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251")),
							types.NewIntraChainOutput(*mock.Eth2BtcOrders[2].ToAssetID, 9, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19255")),
							// re-order
							types.NewIntraChainOutput(*mock.Eth2BtcOrders[2].FromAssetID, 270, mock.Eth2BtcOrders[2].Utxo.ControlProgram),
							// fee
							types.NewIntraChainOutput(*mock.Btc2EthOrders[2].ToAssetID, 41, mock.RewardProgram),
							types.NewIntraChainOutput(*mock.Eth2BtcOrders[2].ToAssetID, 1, mock.RewardProgram),
						},
					}),
				},
			},
			verifyResults: []*bc.TxVerifyResult{{StatusFail: false}},
			wantError:     match.ErrAmountOfFeeOutOfRange,
		},
		{
			desc: "ratio numerator is zero",
			block: &types.Block{
				Transactions: []*types.Tx{
					types.NewTx(types.TxData{
						Inputs:  []*types.TxInput{types.NewSpendInput(nil, *mock.Btc2EthOrders[0].Utxo.SourceID, *mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.Btc2EthOrders[0].Utxo.SourcePos, []byte{0x51})},
						Outputs: []*types.TxOutput{types.NewIntraChainOutput(*mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.MustCreateP2WMCProgram(mock.ETH, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251"), 0, 1))},
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
						Outputs: []*types.TxOutput{types.NewIntraChainOutput(*mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.MustCreateP2WMCProgram(mock.ETH, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251"), 1, 0))},
					}),
				},
			},
			verifyResults: []*bc.TxVerifyResult{{StatusFail: false}},
			wantError:     errRatioOfTradeLessThanZero,
		},
		{
			desc: "want amount is overflow",
			block: &types.Block{
				Transactions: []*types.Tx{
					types.NewTx(types.TxData{
						Inputs:  []*types.TxInput{types.NewSpendInput(nil, *mock.Btc2EthOrders[0].Utxo.SourceID, *mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.Btc2EthOrders[0].Utxo.SourcePos, []byte{0x51})},
						Outputs: []*types.TxOutput{types.NewIntraChainOutput(*mock.Btc2EthOrders[0].FromAssetID, mock.Btc2EthOrders[0].Utxo.Amount, mock.MustCreateP2WMCProgram(mock.ETH, testutil.MustDecodeHexString("0014f928b723999312df4ed51cb275a2644336c19251"), math.MaxInt64, 1))},
					}),
				},
			},
			verifyResults: []*bc.TxVerifyResult{{StatusFail: false}},
			wantError:     errRequestAmountMath,
		},
	}

	for i, c := range cases {
		movCore := &Core{}
		c.block.Height = 3456786543
		if err := movCore.ValidateBlock(c.block, c.verifyResults); err != c.wantError {
			t.Errorf("#%d(%s):validate block want error(%v), got error(%v)", i, c.desc, c.wantError, err)
		}
	}
}

func TestCalcMatchedTxFee(t *testing.T) {
	cases := []struct {
		desc             string
		tx               types.TxData
		maxFeeRate       float64
		wantMatchedTxFee map[bc.AssetID]*matchedTxFee
	}{
		{
			desc:       "fee less than max fee",
			maxFeeRate: 0.05,
			wantMatchedTxFee: map[bc.AssetID]*matchedTxFee{
				mock.ETH: {amount: 11, rewardProgram: mock.RewardProgram},
				mock.BTC: {amount: 1, rewardProgram: mock.RewardProgram},
			},
			tx: mock.MatchedTxs[1].TxData,
		},
		{
			desc:       "fee refund in tx",
			maxFeeRate: 0.05,
			wantMatchedTxFee: map[bc.AssetID]*matchedTxFee{
				mock.ETH: {amount: 25, rewardProgram: mock.RewardProgram},
				mock.BTC: {amount: 1, rewardProgram: mock.RewardProgram},
			},
			tx: mock.MatchedTxs[2].TxData,
		},
		{
			desc:       "no price diff",
			maxFeeRate: 0.05,
			wantMatchedTxFee: map[bc.AssetID]*matchedTxFee{
				mock.ETH: {amount: 1, rewardProgram: mock.RewardProgram},
				mock.BTC: {amount: 1, rewardProgram: mock.RewardProgram},
			},
			tx: mock.MatchedTxs[0].TxData,
		},
	}

	for i, c := range cases {
		gotMatchedTxFee, err := calcFeeAmount(types.NewTx(c.tx))
		if err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(gotMatchedTxFee, c.wantMatchedTxFee) {
			t.Errorf("#%d(%s):fail to caculate matched tx fee, got (%v), want (%v)", i, c.desc, gotMatchedTxFee, c.wantMatchedTxFee)
		}
	}
}

func TestBeforeProposalBlock(t *testing.T) {
	consensus.ActiveNetParams.MovRewardPrograms = []consensus.MovRewardProgram{
		{
			BeginBlock: 0,
			EndBlock:   100,
			Program:    hex.EncodeToString(mock.RewardProgram),
		},
	}

	cases := []struct {
		desc           string
		initOrders     []*common.Order
		gasLeft        int64
		wantMatchedTxs []*types.Tx
	}{
		{
			desc:           "has matched tx, but gas left is zero",
			initOrders:     []*common.Order{mock.Btc2EthOrders[0], mock.Eth2BtcOrders[0]},
			gasLeft:        0,
			wantMatchedTxs: []*types.Tx{},
		},
		{
			desc:           "has one matched tx, and gas is sufficient",
			initOrders:     []*common.Order{mock.Btc2EthOrders[0], mock.Eth2BtcOrders[0]},
			gasLeft:        2000,
			wantMatchedTxs: []*types.Tx{mock.MatchedTxs[1]},
		},
		{
			desc: "has two matched tx, but gas is only enough to pack a matched tx",
			initOrders: []*common.Order{
				mock.Btc2EthOrders[0],
				mock.Btc2EthOrders[1],
				mock.Eth2BtcOrders[2],
			},
			gasLeft:        2000,
			wantMatchedTxs: []*types.Tx{mock.MatchedTxs[2]},
		},
		{
			desc: "has two matched tx, and gas left is sufficient",
			initOrders: []*common.Order{
				mock.Btc2EthOrders[0],
				mock.Btc2EthOrders[1],
				mock.Eth2BtcOrders[2],
			},
			gasLeft:        4000,
			wantMatchedTxs: []*types.Tx{mock.MatchedTxs[2], mock.MatchedTxs[3]},
		},
		{
			desc: "has multiple trade pairs, and gas left is sufficient",
			initOrders: []*common.Order{
				mock.Btc2EthOrders[0],
				mock.Btc2EthOrders[1],
				mock.Eth2BtcOrders[2],
				mock.Eos2EtcOrders[0],
				mock.Etc2EosOrders[0],
			},
			gasLeft:        6000,
			wantMatchedTxs: []*types.Tx{mock.MatchedTxs[2], mock.MatchedTxs[3], mock.MatchedTxs[5]},
		},
	}

	for i, c := range cases {
		testDB := dbm.NewDB("testdb", "leveldb", "temp")
		store := database.NewLevelDBMovStore(testDB)
		if err := store.InitDBState(0, &bc.Hash{}); err != nil {
			t.Fatal(err)
		}

		if err := store.ProcessOrders(c.initOrders, nil, initBlockHeader); err != nil {
			t.Fatal(err)
		}

		movCore := &Core{movStore: store}
		gotMatchedTxs, err := movCore.BeforeProposalBlock(nil, 2, c.gasLeft, func() bool { return false })
		if err != nil {
			t.Fatal(err)
		}

		gotMatchedTxMap := make(map[string]interface{})
		for _, matchedTx := range gotMatchedTxs {
			gotMatchedTxMap[matchedTx.ID.String()] = nil
		}

		wantMatchedTxMap := make(map[string]interface{})
		for _, matchedTx := range c.wantMatchedTxs {
			wantMatchedTxMap[matchedTx.ID.String()] = nil
		}

		if !testutil.DeepEqual(gotMatchedTxMap, wantMatchedTxMap) {
			t.Errorf("#%d(%s):want matched tx(%v) is not equals got matched tx(%v)", i, c.desc, c.wantMatchedTxs, gotMatchedTxs)
		}

		testDB.Close()
		os.RemoveAll("temp")
	}
}

func TestValidateMatchedTxSequence(t *testing.T) {
	cases := []struct {
		desc         string
		initOrders   []*common.Order
		transactions []*types.Tx
		wantError    error
	}{
		{
			desc:         "both db orders and transactions is empty",
			initOrders:   []*common.Order{},
			transactions: []*types.Tx{},
			wantError:    nil,
		},
		{
			desc:         "existing matched orders in db, and transactions is empty",
			initOrders:   []*common.Order{mock.Btc2EthOrders[0], mock.Eth2BtcOrders[0]},
			transactions: []*types.Tx{},
			wantError:    nil,
		},
		{
			desc:         "db orders is empty, but transactions has matched tx",
			initOrders:   []*common.Order{},
			transactions: []*types.Tx{mock.MatchedTxs[1]},
			wantError:    errNotMatchedOrder,
		},
		{
			desc:         "existing matched orders in db, and corresponding matched tx in transactions",
			initOrders:   []*common.Order{mock.Btc2EthOrders[0], mock.Eth2BtcOrders[0]},
			transactions: []*types.Tx{mock.MatchedTxs[1]},
			wantError:    nil,
		},
		{
			desc:         "package matched tx, one order from db, and the another order from transactions",
			initOrders:   []*common.Order{mock.Btc2EthOrders[0]},
			transactions: []*types.Tx{mock.Eth2BtcMakerTxs[0], mock.MatchedTxs[10]},
			wantError:    nil,
		},
		{
			desc:         "two matched txs use the same orders",
			initOrders:   []*common.Order{mock.Btc2EthOrders[0], mock.Eth2BtcOrders[0]},
			transactions: []*types.Tx{mock.MatchedTxs[1], mock.MatchedTxs[1]},
			wantError:    errNotMatchedOrder,
		},
		{
			desc: "existing two matched orders in db, and only one corresponding matched tx in transactions",
			initOrders: []*common.Order{
				mock.Btc2EthOrders[3], mock.Eth2BtcOrders[2],
				mock.Btc2EthOrders[0], mock.Eth2BtcOrders[0],
			},
			transactions: []*types.Tx{mock.MatchedTxs[8]},
			wantError:    nil,
		},
		{
			desc: "existing two matched orders in db, and the sequence of match txs in incorrect",
			initOrders: []*common.Order{
				mock.Btc2EthOrders[3], mock.Eth2BtcOrders[2],
				mock.Btc2EthOrders[0], mock.Eth2BtcOrders[0],
			},
			transactions: []*types.Tx{mock.MatchedTxs[1], mock.MatchedTxs[8]},
			wantError:    errSpendOutputIDIsIncorrect,
		},
		{
			desc:         "matched tx and orders from packaged transactions",
			initOrders:   []*common.Order{},
			transactions: []*types.Tx{mock.Btc2EthMakerTxs[0], mock.Eth2BtcMakerTxs[1], mock.MatchedTxs[4]},
			wantError:    nil,
		},
		{
			desc:         "package the matched tx first, then package match orders",
			initOrders:   []*common.Order{},
			transactions: []*types.Tx{mock.MatchedTxs[4], mock.Btc2EthMakerTxs[0], mock.Eth2BtcMakerTxs[1]},
			wantError:    errNotMatchedOrder,
		},
		{
			desc:         "cancel order in transactions",
			initOrders:   []*common.Order{mock.Btc2EthOrders[0], mock.Eth2BtcOrders[0]},
			transactions: []*types.Tx{mock.Btc2EthCancelTxs[0], mock.MatchedTxs[1]},
			wantError:    errNotMatchedOrder,
		},
		{
			desc:         "package cancel order after match tx",
			initOrders:   []*common.Order{mock.Btc2EthOrders[0], mock.Eth2BtcOrders[0]},
			transactions: []*types.Tx{mock.MatchedTxs[1], mock.Btc2EthCancelTxs[0]},
			wantError:    nil,
		},
		{
			desc: "package matched txs of different trade pairs",
			initOrders: []*common.Order{
				mock.Btc2EthOrders[0], mock.Eth2BtcOrders[0],
				mock.Eos2EtcOrders[0], mock.Etc2EosOrders[0],
			},
			transactions: []*types.Tx{mock.MatchedTxs[1], mock.MatchedTxs[9]},
			wantError:    nil,
		},
		{
			desc: "package matched txs of different trade pairs in different sequence",
			initOrders: []*common.Order{
				mock.Btc2EthOrders[0], mock.Eth2BtcOrders[0],
				mock.Eos2EtcOrders[0], mock.Etc2EosOrders[0],
			},
			transactions: []*types.Tx{mock.MatchedTxs[9], mock.MatchedTxs[1]},
			wantError:    nil,
		},
		{
			desc:         "package partial matched tx from db orders, and the re-pending order continue to match",
			initOrders:   []*common.Order{mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Eth2BtcOrders[2]},
			transactions: []*types.Tx{mock.MatchedTxs[2], mock.MatchedTxs[3]},
			wantError:    nil,
		},
		{
			desc:         "cancel the re-pending order",
			initOrders:   []*common.Order{mock.Btc2EthOrders[0], mock.Btc2EthOrders[1], mock.Eth2BtcOrders[2]},
			transactions: []*types.Tx{mock.MatchedTxs[2], mock.Btc2EthCancelTxs[1], mock.MatchedTxs[3]},
			wantError:    errNotMatchedOrder,
		},
	}

	for i, c := range cases {
		testDB := dbm.NewDB("testdb", "leveldb", "temp")
		store := database.NewLevelDBMovStore(testDB)
		if err := store.InitDBState(0, &bc.Hash{}); err != nil {
			t.Fatal(err)
		}

		if err := store.ProcessOrders(c.initOrders, nil, initBlockHeader); err != nil {
			t.Fatal(err)
		}

		movCore := &Core{movStore: store}
		if err := movCore.validateMatchedTxSequence(c.transactions); err != c.wantError {
			t.Errorf("#%d(%s):wanet error(%v), got error(%v)", i, c.desc, c.wantError, err)
		}

		testDB.Close()
		os.RemoveAll("temp")
	}
}

type testFun func(movCore *Core, block *types.Block) error

func applyBlock(movCore *Core, block *types.Block) error {
	return movCore.ApplyBlock(block)
}

func detachBlock(movCore *Core, block *types.Block) error {
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
