package test

import (
	"os"
	"testing"

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

func TestRollback(t *testing.T) {
	cases := []struct {
		desc                        string
		movStartHeight              uint64
		RoundVoteBlockNums          uint64
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
			desc:               "rollback from height 1 to 0",
			movStartHeight:     10,
			RoundVoteBlockNums: 1200,
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
		{
			desc:               "rollback from height 2 to 0",
			movStartHeight:     10,
			RoundVoteBlockNums: 1200,
			beforeBestBlockHeader: &types.BlockHeader{
				Height:            2,
				PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
			},
			wantBestBlockHeader: &types.BlockHeader{
				Height: 0,
			},
			beforeLastIrrBlockHeader: &types.BlockHeader{
				Height:            2,
				PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
			},
			wantLastIrrBlockHeader: &types.BlockHeader{
				Height: 0,
			},
			beforeUtxoViewPoint: &state.UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					testutil.MustDecodeHash("afee09925bea1695424450a91ad082a378f20534627fa5cb63f036846347ee08"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 2, Spent: true},
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
				{
					BlockHeader: types.BlockHeader{
						Height:            2,
						PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 3000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 2500, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
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
						"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100004500,
						"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 100002000,
					},
					BlockHash:      testutil.MustDecodeHash("699d3f59d4afe7eea85df31814628d7d34ace7f5e76d6c9ebf4c54482d2cd333"),
					BlockHeight:    2,
					CoinbaseReward: map[string]uint64{"0001": consensus.BlockSubsidy(2) + 10000000000},
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
		{
			desc:               "rollback from height 2 to 1",
			movStartHeight:     10,
			RoundVoteBlockNums: 1200,
			beforeBestBlockHeader: &types.BlockHeader{
				Height:            2,
				PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
			},
			wantBestBlockHeader: &types.BlockHeader{
				Height:            1,
				PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
			},
			beforeLastIrrBlockHeader: &types.BlockHeader{
				Height:            2,
				PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
			},
			wantLastIrrBlockHeader: &types.BlockHeader{
				Height:            1,
				PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
			},
			beforeUtxoViewPoint: &state.UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					testutil.MustDecodeHash("afee09925bea1695424450a91ad082a378f20534627fa5cb63f036846347ee08"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 2, Spent: true},
					testutil.MustDecodeHash("51f538be366172bed5359a016dce26b952024c9607caf6af609ad723982c2e06"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
					testutil.MustDecodeHash("e2370262a129b90174195a76c298d872a56af042eae17657e154bcc46d41b3ba"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
				},
			},
			rollbackToTargetHeight: 1,
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
				{
					BlockHeader: types.BlockHeader{
						Height:            2,
						PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 3000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 2500, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
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
			beforeStoredConsensusResult: []*state.ConsensusResult{
				{
					Seq: 1,
					NumOfVote: map[string]uint64{
						"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100004500,
						"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 100002000,
					},
					BlockHash:      testutil.MustDecodeHash("699d3f59d4afe7eea85df31814628d7d34ace7f5e76d6c9ebf4c54482d2cd333"),
					BlockHeight:    2,
					CoinbaseReward: map[string]uint64{"0001": consensus.BlockSubsidy(1) + consensus.BlockSubsidy(2) + 500},
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
				Seq: 1,
				NumOfVote: map[string]uint64{
					"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100002000,
					"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 100002000,
				},
				BlockHash:      testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
				BlockHeight:    1,
				CoinbaseReward: map[string]uint64{"0001": consensus.BlockSubsidy(1)},
			},
			wantUtxoViewPoint: &state.UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					testutil.MustDecodeHash("51f538be366172bed5359a016dce26b952024c9607caf6af609ad723982c2e06"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
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
		{
			desc:               "rollback from height 2 to 1, RoundVoteBlockNums is 2",
			movStartHeight:     10,
			RoundVoteBlockNums: 2,
			beforeBestBlockHeader: &types.BlockHeader{
				Height:            2,
				PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
			},
			wantBestBlockHeader: &types.BlockHeader{
				Height:            1,
				PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
			},
			beforeLastIrrBlockHeader: &types.BlockHeader{
				Height:            2,
				PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
			},
			wantLastIrrBlockHeader: &types.BlockHeader{
				Height:            1,
				PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
			},
			beforeUtxoViewPoint: &state.UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					testutil.MustDecodeHash("afee09925bea1695424450a91ad082a378f20534627fa5cb63f036846347ee08"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 2, Spent: true},
					testutil.MustDecodeHash("51f538be366172bed5359a016dce26b952024c9607caf6af609ad723982c2e06"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
					testutil.MustDecodeHash("e2370262a129b90174195a76c298d872a56af042eae17657e154bcc46d41b3ba"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
				},
			},
			rollbackToTargetHeight: 1,
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
				{
					BlockHeader: types.BlockHeader{
						Height:            2,
						PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 3000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 2500, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
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
			beforeStoredConsensusResult: []*state.ConsensusResult{
				{
					Seq: 1,
					NumOfVote: map[string]uint64{
						"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100004500,
						"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 100002000,
					},
					BlockHash:      testutil.MustDecodeHash("699d3f59d4afe7eea85df31814628d7d34ace7f5e76d6c9ebf4c54482d2cd333"),
					BlockHeight:    2,
					CoinbaseReward: map[string]uint64{"0001": consensus.BlockSubsidy(1) + consensus.BlockSubsidy(2) + 500},
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
				Seq: 1,
				NumOfVote: map[string]uint64{
					"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100002000,
					"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 100002000,
				},
				BlockHash:      testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
				BlockHeight:    1,
				CoinbaseReward: map[string]uint64{"0001": consensus.BlockSubsidy(1)},
			},
			wantUtxoViewPoint: &state.UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					testutil.MustDecodeHash("51f538be366172bed5359a016dce26b952024c9607caf6af609ad723982c2e06"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
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
		{
			desc:               "rollback from height 3 to 1, RoundVoteBlockNums is 2",
			movStartHeight:     10,
			RoundVoteBlockNums: 2,
			beforeBestBlockHeader: &types.BlockHeader{
				Height:            3,
				PreviousBlockHash: testutil.MustDecodeHash("699d3f59d4afe7eea85df31814628d7d34ace7f5e76d6c9ebf4c54482d2cd333"),
			},
			wantBestBlockHeader: &types.BlockHeader{
				Height:            1,
				PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
			},
			beforeLastIrrBlockHeader: &types.BlockHeader{
				Height:            3,
				PreviousBlockHash: testutil.MustDecodeHash("699d3f59d4afe7eea85df31814628d7d34ace7f5e76d6c9ebf4c54482d2cd333"),
			},
			wantLastIrrBlockHeader: &types.BlockHeader{
				Height:            1,
				PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
			},
			beforeUtxoViewPoint: &state.UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					testutil.MustDecodeHash("5b4d53fbc2a489847f34dd0e0c085797fe7cf0a3a9a2f3231d11bdad16dea2be"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 3, Spent: true},
					testutil.MustDecodeHash("4c2b719d10fc6b9c2a7c343491ddd8c0d6bd57f9c6680bfda557689c182cf685"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 2, Spent: true},
					testutil.MustDecodeHash("9fb6f213e3130810e755675707d0e9870c79a91c575638a580fae65568ca9e99"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
					testutil.MustDecodeHash("3d1617908e624a2042c23be4f671b261d5b8a2a61b8421ee6a702c6e071428a8"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
				},
			},
			rollbackToTargetHeight: 1,
			beforeStoredBlocks: []*types.Block{
				{
					BlockHeader: types.BlockHeader{
						Height: 0,
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						}),
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 100000000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 100000000, []byte{0, 1}, testutil.MustDecodeHexString("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67")),
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
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						}),
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 200000000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 200000000-2000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
							},
						}),
					},
				},
				{
					BlockHeader: types.BlockHeader{
						Height:            2,
						PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{
								types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51}),
							},
						}),
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 300000000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 250000000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
							},
						}),
					},
				},
				{
					BlockHeader: types.BlockHeader{
						Height:            3,
						PreviousBlockHash: testutil.MustDecodeHash("699d3f59d4afe7eea85df31814628d7d34ace7f5e76d6c9ebf4c54482d2cd333"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{
								types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51}),
								types.NewIntraChainOutput(bc.AssetID{}, consensus.BlockSubsidy(1)+consensus.BlockSubsidy(2)+50002000, []byte{0x51}),
							},
						}),
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 400000000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 160000000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
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
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						}),
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 100000000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 100000000, []byte{0, 1}, testutil.MustDecodeHexString("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67")),
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
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						}),
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 200000000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 200000000-2000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
							},
						}),
					},
				},
			},
			beforeStoredConsensusResult: []*state.ConsensusResult{
				{
					Seq: 2,
					NumOfVote: map[string]uint64{
						"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 609998000,
						"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 200000000,
					},
					BlockHash:      testutil.MustDecodeHash("0c1cd1c0a6e6161f437c382cca21ce28921234ed7c4f252f7e4bbc9a523b74ac"),
					BlockHeight:    3,
					CoinbaseReward: map[string]uint64{"51": consensus.BlockSubsidy(3) + 240000000},
				},
				{
					Seq: 1,
					NumOfVote: map[string]uint64{
						"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 449998000,
						"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 200000000,
					},
					BlockHash:      testutil.MustDecodeHash("699d3f59d4afe7eea85df31814628d7d34ace7f5e76d6c9ebf4c54482d2cd333"),
					BlockHeight:    2,
					CoinbaseReward: map[string]uint64{"51": consensus.BlockSubsidy(1) + consensus.BlockSubsidy(2) + 50002000},
				},
				{
					Seq: 0,
					NumOfVote: map[string]uint64{
						"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100000000,
						"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 200000000,
					},
					BlockHash:      testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
					BlockHeight:    0,
					CoinbaseReward: map[string]uint64{},
				},
			},
			wantBestConsensusResult: &state.ConsensusResult{
				Seq: 1,
				NumOfVote: map[string]uint64{
					"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100000000 + 100000000 - 2000,
					"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 200000000,
				},
				BlockHash:      testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
				BlockHeight:    1,
				CoinbaseReward: map[string]uint64{"51": consensus.BlockSubsidy(1) + 2000},
			},
			wantUtxoViewPoint: &state.UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					testutil.MustDecodeHash("9fb6f213e3130810e755675707d0e9870c79a91c575638a580fae65568ca9e99"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
					testutil.MustDecodeHash("3d1617908e624a2042c23be4f671b261d5b8a2a61b8421ee6a702c6e071428a8"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
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
		{
			desc:               "rollback from height 3 to 2, RoundVoteBlockNums is 2",
			movStartHeight:     10,
			RoundVoteBlockNums: 2,
			beforeBestBlockHeader: &types.BlockHeader{
				Height:            3,
				PreviousBlockHash: testutil.MustDecodeHash("699d3f59d4afe7eea85df31814628d7d34ace7f5e76d6c9ebf4c54482d2cd333"),
			},
			wantBestBlockHeader: &types.BlockHeader{
				Height:            2,
				PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
			},
			beforeLastIrrBlockHeader: &types.BlockHeader{
				Height:            3,
				PreviousBlockHash: testutil.MustDecodeHash("699d3f59d4afe7eea85df31814628d7d34ace7f5e76d6c9ebf4c54482d2cd333"),
			},
			wantLastIrrBlockHeader: &types.BlockHeader{
				Height:            2,
				PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
			},
			beforeUtxoViewPoint: &state.UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					testutil.MustDecodeHash("5b4d53fbc2a489847f34dd0e0c085797fe7cf0a3a9a2f3231d11bdad16dea2be"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 3, Spent: true},
					testutil.MustDecodeHash("4c2b719d10fc6b9c2a7c343491ddd8c0d6bd57f9c6680bfda557689c182cf685"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 2, Spent: true},
					testutil.MustDecodeHash("9fb6f213e3130810e755675707d0e9870c79a91c575638a580fae65568ca9e99"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
					testutil.MustDecodeHash("3d1617908e624a2042c23be4f671b261d5b8a2a61b8421ee6a702c6e071428a8"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
				},
			},
			rollbackToTargetHeight: 2,
			beforeStoredBlocks: []*types.Block{
				{
					BlockHeader: types.BlockHeader{
						Height: 0,
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						}),
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 100000000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 100000000, []byte{0, 1}, testutil.MustDecodeHexString("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67")),
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
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						}),
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 200000000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 200000000-2000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
							},
						}),
					},
				},
				{
					BlockHeader: types.BlockHeader{
						Height:            2,
						PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{
								types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51}),
							},
						}),
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 300000000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 250000000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
							},
						}),
					},
				},
				{
					BlockHeader: types.BlockHeader{
						Height:            3,
						PreviousBlockHash: testutil.MustDecodeHash("699d3f59d4afe7eea85df31814628d7d34ace7f5e76d6c9ebf4c54482d2cd333"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{
								types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51}),
								types.NewIntraChainOutput(bc.AssetID{}, consensus.BlockSubsidy(1)+consensus.BlockSubsidy(2)+50002000, []byte{0x51}),
							},
						}),
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 400000000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 160000000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
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
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						}),
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 100000000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 100000000, []byte{0, 1}, testutil.MustDecodeHexString("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67")),
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
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						}),
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 200000000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 200000000-2000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
							},
						}),
					},
				},
				{
					BlockHeader: types.BlockHeader{
						Height:            2,
						PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
					},
					Transactions: []*types.Tx{
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{
								types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51}),
							},
						}),
						types.NewTx(types.TxData{
							Inputs: []*types.TxInput{
								types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 300000000, 0, []byte{0, 1}),
							},
							Outputs: []*types.TxOutput{
								types.NewVoteOutput(*consensus.BTMAssetID, 250000000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
							},
						}),
					},
				},
			},
			beforeStoredConsensusResult: []*state.ConsensusResult{
				{
					Seq: 2,
					NumOfVote: map[string]uint64{
						"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 609998000,
						"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 200000000,
					},
					BlockHash:      testutil.MustDecodeHash("0c1cd1c0a6e6161f437c382cca21ce28921234ed7c4f252f7e4bbc9a523b74ac"),
					BlockHeight:    3,
					CoinbaseReward: map[string]uint64{"51": consensus.BlockSubsidy(3) + 240000000},
				},
				{
					Seq: 1,
					NumOfVote: map[string]uint64{
						"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 449998000,
						"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 200000000,
					},
					BlockHash:      testutil.MustDecodeHash("699d3f59d4afe7eea85df31814628d7d34ace7f5e76d6c9ebf4c54482d2cd333"),
					BlockHeight:    2,
					CoinbaseReward: map[string]uint64{"51": consensus.BlockSubsidy(1) + consensus.BlockSubsidy(2) + 50002000},
				},
				{
					Seq: 0,
					NumOfVote: map[string]uint64{
						"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100000000,
						"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 200000000,
					},
					BlockHash:      testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
					BlockHeight:    0,
					CoinbaseReward: map[string]uint64{},
				},
			},
			wantBestConsensusResult: &state.ConsensusResult{
				Seq: 1,
				NumOfVote: map[string]uint64{
					"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100000000 + 100000000 - 2000 + 250000000,
					"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 200000000,
				},
				BlockHash:      testutil.MustDecodeHash("699d3f59d4afe7eea85df31814628d7d34ace7f5e76d6c9ebf4c54482d2cd333"),
				BlockHeight:    2,
				CoinbaseReward: map[string]uint64{"51": consensus.BlockSubsidy(1) + consensus.BlockSubsidy(2) + 50002000},
			},
			wantUtxoViewPoint: &state.UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					testutil.MustDecodeHash("9fb6f213e3130810e755675707d0e9870c79a91c575638a580fae65568ca9e99"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
					testutil.MustDecodeHash("3d1617908e624a2042c23be4f671b261d5b8a2a61b8421ee6a702c6e071428a8"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
					testutil.MustDecodeHash("4c2b719d10fc6b9c2a7c343491ddd8c0d6bd57f9c6680bfda557689c182cf685"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 2, Spent: true},
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
		os.RemoveAll("block_db")
		os.RemoveAll("mov_db")
		consensus.ActiveNetParams.RoundVoteBlockNums = c.RoundVoteBlockNums

		movDB := dbm.NewDB("mov_db", "leveldb", "mov_db")
		movStore := movDatabase.NewLevelDBMovStore(movDB)

		movCore := mov.NewMovCoreWithDB(movStore, c.movStartHeight)
		blockDB := dbm.NewDB("block_db", "leveldb", "block_db")
		store := database.NewStore(blockDB)

		mainChainBlockHeaders := []*types.BlockHeader{}
		for _, block := range c.beforeStoredBlocks {
			status := bc.NewTransactionStatus()
			for index := range block.Transactions {
				status.SetStatus(index, false)
			}
			store.SaveBlock(block, status)
			mainChainBlockHeaders = append(mainChainBlockHeaders, &block.BlockHeader)
		}

		if err := store.SaveChainStatus(c.beforeBestBlockHeader, c.beforeLastIrrBlockHeader, mainChainBlockHeaders, c.beforeUtxoViewPoint, c.beforeStoredConsensusResult); err != nil {
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

		transOldTx := []*bc.Tx{}
		for _, block := range c.wantStoredBlocks {
			hash := block.Hash()
			for _, tx := range block.Transactions {
				transOldTx = append(transOldTx, tx.Tx)
			}
			gotBlock, err := store.GetBlock(&hash)
			if err != nil {
				t.Fatal(err)
			}

			if !testutil.DeepEqual(block.BlockHeader, gotBlock.BlockHeader) {
				t.Fatalf("this block height %d should existed!", block.Height)
			}
		}

		nowUtxoViewPoint := state.NewUtxoViewpoint()
		if err = store.GetTransactionsUtxo(nowUtxoViewPoint, transOldTx); err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(nowUtxoViewPoint, c.wantUtxoViewPoint) {
			t.Fatal(err)
		}

		blockDB.Close()
		os.RemoveAll("block_db")
		movDB.Close()
		os.RemoveAll("mov_db")

	}
}
