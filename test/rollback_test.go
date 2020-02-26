package test

import (
	"os"
	"testing"

	"github.com/bytom/vapor/application/mov"
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

func TestRollback(t *testing.T) {
	cases := []struct {
		desc                        string
		beforeBestBlockHeader       *types.BlockHeader
		beforeLastIrrBlockHeader    *types.BlockHeader
		beforeUtxoViewPoint         *state.UtxoViewpoint
		beforeStoredBlocks          []*types.Block
		beforeStoredConsensusResult []*state.ConsensusResult
		wantBestBlockHeader         *types.BlockHeader
		wantLastIrrBlockHeader      *types.BlockHeader
		wantBestConsensusResult     *state.ConsensusResult
		rollbackToTargetHeight      uint64
	}{
		{
			desc: "rollback from height 1 to 0",
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
					testutil.MustDecodeHash("c094bdfd925b4f357a7cb373f8b9ec001181c9217fd7de8219ea1163a1bee93f"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: false},
					testutil.MustDecodeHash("82dc360aaee03b2d42f964befdaf8ab36930e1578d14547da9bd7d23062ecf3c"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
				},
			},
			rollbackToTargetHeight: 0,
			beforeStoredBlocks: []*types.Block{
				{
					BlockHeader: types.BlockHeader{
						Height: 0,
					},
					Transactions: []*types.Tx{
						{
							TxData: types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 1000, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 1000, []byte{0, 1}, testutil.MustDecodeHexString("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67")),
								},
							},
						},
					},
				},
				{
					BlockHeader: types.BlockHeader{
						Height:            1,
						PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
					},
					Transactions: []*types.Tx{
						{
							TxData: types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 2000, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 2000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
								},
							},
						},
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
		},
	}

	for _, c := range cases {
		movDB := dbm.NewDB("mov_db", "leveldb", "mov_db")
		movCore := mov.NewMovCoreWithDB(movDB, 10)
		blockDB := dbm.NewDB("block_db", "leveldb", "block_db")
		store := database.NewStore(blockDB)

		mainChainBlocks := []*types.BlockHeader{}
		for _, block := range c.beforeStoredBlocks {
			newTrans := []*types.Tx{}
			status := bc.NewTransactionStatus()
			for index, tx := range block.Transactions {
				status.SetStatus(index, false)
				tx := &types.Tx{TxData: tx.TxData, Tx: types.MapTx(&tx.TxData)}
				newTrans = append(newTrans, tx)
			}

			store.SaveBlock(block, status)
			mainChainBlocks = append(mainChainBlocks, &block.BlockHeader)
		}

		if err := store.SaveChainStatus(c.beforeBestBlockHeader, c.beforeLastIrrBlockHeader, mainChainBlocks, c.beforeUtxoViewPoint, c.beforeStoredConsensusResult); err != nil {
			t.Fatal(err)
		}

		chain, err := protocol.NewChain(store, nil, []protocol.Protocoler{movCore}, nil)
		if err != nil {
			t.Fatal(err)
		}

		if err := chain.Rollback(c.rollbackToTargetHeight); err != nil {
			t.Fatal(err)
		}

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

		blockDB.Close()
		os.RemoveAll("block_db")
		movDB.Close()
		os.RemoveAll("mov_db")
	}
}
