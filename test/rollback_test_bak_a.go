package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytom/vapor/application/mov"
	movDatabase "github.com/bytom/vapor/application/mov/database"
	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/database"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/database/storage"
	"github.com/bytom/vapor/protocol"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
	"github.com/bytom/vapor/testutil"
)

func compareDBSame(t *testing.T, dbA dbm.DB, dbB dbm.DB) bool {
	iterA := dbA.Iterator()
	iterB := dbB.Iterator()

	for iterA.Next() && iterB.Next() {
		require.Equal(t, iterA.Key(), iterB.Key())
		require.Equal(t, iterA.Value(), iterB.Value())
	}

	if iterA.Next() || iterB.Next() {
		t.Fatalf("why iterator is not finished")
	}

	return true
}

func ATestSmall(t *testing.T) {
	wantStoredBlocks := []*types.Block{
		{
			BlockHeader: types.BlockHeader{
				Height: 0,
			},
			Transactions: []*types.Tx{
				types.NewTx(types.TxData{
					Inputs: []*types.TxInput{
						types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 1000, 0, []byte{0, 1}),
					},
					Outputs: []*types.TxOutput{
						types.NewVoteOutput(*consensus.BTMAssetID, 1000, []byte{0, 1}, testutil.MustDecodeHexString("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67")),
					},
				}),
			},
		},
		{
			BlockHeader: types.BlockHeader{
				Height: 1,
			},
			Transactions: []*types.Tx{
				types.NewTx(types.TxData{
					Inputs: []*types.TxInput{
						types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 1000, 0, []byte{0, 1}),
					},
					Outputs: []*types.TxOutput{
						types.NewVoteOutput(*consensus.BTMAssetID, 1000, []byte{0, 1}, testutil.MustDecodeHexString("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67")),
					},
				}),
			},
		},
	}

	dbA := dbm.NewDB("dba", "leveldb", "dba")
	storeA := database.NewStore(dbA)
	status := bc.NewTransactionStatus()
	status.SetStatus(0, false)

	storeA.SaveBlock(wantStoredBlocks[0], status)

	//hash := wantStoredBlocks[0].Hash()
	// block, _ := storeA.GetBlock(&hash)
	// fmt.Println("!!!!!!!!!!", block)
	// fmt.Println("????????", wantStoredBlocks[0])

	//fmt.Println("amazing!")

	dbB := dbm.NewDB("dbb", "leveldb", "dbb")
	storeB := database.NewStore(dbB)
	storeB.SaveBlock(wantStoredBlocks[0], status)
	storeB.SaveBlock(wantStoredBlocks[1], status)

	storeB.DeleteBlock(wantStoredBlocks[1])

	compareDBSame(t, dbA, dbB)
	dbA.Close()
	dbB.Close()
	os.RemoveAll("dba")
	os.RemoveAll("dbb")
}

func ATestRollback(t *testing.T) {
	// 1-->0
	// 2-->0
	// 2-->1
	// 1200 个区块回滚 , 1201--> 1199 , 1201-->1200, 1200->1199
	cases := []struct {
		desc                        string
		movStartHeight              uint64
		beforeBestBlockHeader       *types.BlockHeader
		beforeLastIrrBlockHeader    *types.BlockHeader
		beforeUtxoViewPoint         *state.UtxoViewpoint
		beforeStoredBlocks          []*types.Block
		beforeStoredConsensusResult []*state.ConsensusResult
		wantStoredBlocks            []*types.Block
		wantBestBlockHeader         *types.BlockHeader
		wantLastIrrBlockHeader      *types.BlockHeader
		wantBestConsensusResult     *state.ConsensusResult
		wantUtxoViewPoint           *state.UtxoViewpoint
		wantStoredConsensusResult   []*state.ConsensusResult
		rollbackToTargetHeight      uint64
	}{
		{
			desc:           "rollback from height 1 to 0",
			movStartHeight: 10,
			beforeBestBlockHeader: &types.BlockHeader{
				Height:            1,
				PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
			},
			wantBestBlockHeader: &types.BlockHeader{
				Height: 0,
			},
			beforeLastIrrBlockHeader: &types.BlockHeader{
				Height:            1,
				PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
			},
			wantLastIrrBlockHeader: &types.BlockHeader{
				Height: 0,
			},
			beforeUtxoViewPoint: &state.UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					testutil.MustDecodeHash("51f538be366172bed5359a016dce26b952024c9607caf6af609ad723982c2e06"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
					testutil.MustDecodeHash("e2370262a129b90174195a76c298d872a56af042eae17657e154bcc46d41b3ba"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
				},
			},
			rollbackToTargetHeight: 0,
			beforeStoredBlocks: []*types.Block{
				{
					BlockHeader: types.BlockHeader{
						Height: 0,
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 1000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 1000, []byte{0, 1}, testutil.MustDecodeHexString("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67")),
							},
						}),
					},
				},
				{
					BlockHeader: types.BlockHeader{
						Height:            1,
						PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 2000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 2000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
							},
						}),
					},
				},
			},
			wantStoredBlocks: []*types.Block{
				{
					BlockHeader: types.BlockHeader{
						Height: 0,
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 1000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 1000, []byte{0, 1}, testutil.MustDecodeHexString("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67")),
							},
						}),
					},
				},
			},
			beforeStoredConsensusResult: []*state.ConsensusResult{
				{
					Seq: 1,
					NumOfVote: map[string]uint64{
						"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100002000,
						"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 100002000,
					},
					BlockHash:      testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
					BlockHeight:    1,
					CoinbaseReward: map[string]uint64{"0001": consensus.BlockSubsidy(1) + 10000000000},
				},
				{
					Seq: 0,
					NumOfVote: map[string]uint64{
						"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100000000,
						"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 100002000,
					},
					BlockHash:      testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
					BlockHeight:    0,
					CoinbaseReward: map[string]uint64{},
				},
			},
			wantBestConsensusResult: &state.ConsensusResult{
				Seq: 0,
				NumOfVote: map[string]uint64{
					"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100000000,
					"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 100002000,
				},
				BlockHash:      testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
				BlockHeight:    0,
				CoinbaseReward: map[string]uint64{},
			},
			wantUtxoViewPoint: &state.UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					testutil.MustDecodeHash("51f538be366172bed5359a016dce26b952024c9607caf6af609ad723982c2e06"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: false},
					testutil.MustDecodeHash("e2370262a129b90174195a76c298d872a56af042eae17657e154bcc46d41b3ba"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
				},
			},
			wantStoredConsensusResult: []*state.ConsensusResult{
				{
					Seq: 0,
					NumOfVote: map[string]uint64{
						"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100000000,
						"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 100002000,
					},
					BlockHash:      testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
					BlockHeight:    0,
					CoinbaseReward: map[string]uint64{},
				},
			},
		},
	}

	for _, c := range cases {
		movDB := dbm.NewDB("mov_db", "leveldb", "mov_db")
		movStore := movDatabase.NewLevelDBMovStore(movDB)

		movCore := mov.NewMovCoreWithDB(movStore, c.movStartHeight)
		blockDB := dbm.NewDB("block_db", "leveldb", "block_db")
		store := database.NewStore(blockDB)

		compareDB := dbm.NewDB("compare_block_db", "leveldb", "compare_block_db")
		compareStore := database.NewStore(compareDB)

		mainChainBlockHeaders := []*types.BlockHeader{}
		for _, block := range c.beforeStoredBlocks {
			trans := block.Transactions
			for _, tx := range trans {
				for _, prevout := range tx.SpentOutputIDs {
					fmt.Println(prevout.String())
				}
			}

			status := bc.NewTransactionStatus()
			for index := range block.Transactions {
				status.SetStatus(index, false)
			}
			store.SaveBlock(block, status)

			mainChainBlockHeaders = append(mainChainBlockHeaders, &block.BlockHeader)
		}

		wantMainChainBlockHeaders := []*types.BlockHeader{}
		for _, block := range c.wantStoredBlocks {
			status := bc.NewTransactionStatus()
			for index := range block.Transactions {
				status.SetStatus(index, false)
			}
			compareStore.SaveBlock(block, status)

			wantMainChainBlockHeaders = append(wantMainChainBlockHeaders, &block.BlockHeader)
		}

		if err := store.SaveChainStatus(c.beforeBestBlockHeader, c.beforeLastIrrBlockHeader, mainChainBlockHeaders, c.beforeUtxoViewPoint, c.beforeStoredConsensusResult); err != nil {
			t.Fatal(err)
		}

		if err := compareStore.SaveChainStatus(c.wantBestBlockHeader, c.wantLastIrrBlockHeader, wantMainChainBlockHeaders, c.wantUtxoViewPoint, c.wantStoredConsensusResult); err != nil {
			t.Fatal(err)
		}

		chain, err := protocol.NewChain(store, nil, []protocol.Protocoler{movCore}, nil)
		if err != nil {
			t.Fatal(err)
		}

		if err := chain.Rollback(c.rollbackToTargetHeight); err != nil {
			t.Fatal(err)
		}

		hash := testutil.MustDecodeHash("e2370262a129b90174195a76c298d872a56af042eae17657e154bcc46d41b3ba")
		utxo, err := store.GetUtxo(&hash)
		fmt.Println("store e2370262a129b90174195a76c298d872a56af042eae17657e154bcc46d41b3ba", utxo, err)

		hash = testutil.MustDecodeHash("51f538be366172bed5359a016dce26b952024c9607caf6af609ad723982c2e06")
		utxo, err = store.GetUtxo(&hash)
		fmt.Println("store 51f538be366172bed5359a016dce26b952024c9607caf6af609ad723982c2e06", utxo, err)

		hash = testutil.MustDecodeHash("e2370262a129b90174195a76c298d872a56af042eae17657e154bcc46d41b3ba")
		utxo, err = compareStore.GetUtxo(&hash)
		fmt.Println("compareStore e2370262a129b90174195a76c298d872a56af042eae17657e154bcc46d41b3ba", utxo, err)

		hash = testutil.MustDecodeHash("51f538be366172bed5359a016dce26b952024c9607caf6af609ad723982c2e06")
		utxo, err = compareStore.GetUtxo(&hash)
		fmt.Println("compareStore 51f538be366172bed5359a016dce26b952024c9607caf6af609ad723982c2e06", utxo, err)

		if !testutil.DeepEqual(chain.LastIrreversibleHeader(), c.wantLastIrrBlockHeader) {
			t.Fatalf("lastIrrBlockHeader is not right!")
		}

		if !testutil.DeepEqual(chain.BestBlockHeader(), c.wantBestBlockHeader) {
			t.Fatalf("wantBestBlockHeader is not right!")
		}

		nowConsensusResult, err := chain.GetConsensusResultByHash(chain.BestBlockHash())
		if err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(nowConsensusResult, c.wantBestConsensusResult) {
			t.Fatalf("wantBestConsensusResult is not right!")
		}

		if !compareDBSame(t, blockDB, compareDB) {
			t.Fatalf("the db is not same")
		}

		for _, block := range c.wantStoredBlocks {
			hash := block.Hash()
			gotBlock, err := store.GetBlock(&hash)
			if err != nil {
				t.Fatal(err)
			}

			if !testutil.DeepEqual(block.BlockHeader, gotBlock.BlockHeader) {
				t.Fatalf("this block height %d should existed!", block.Height)
			}
		}

		blockDB.Close()
		os.RemoveAll("block_db")
		movDB.Close()
		os.RemoveAll("mov_db")

		compareDB.Close()
		os.RemoveAll("compare_block_db")
	}
}
