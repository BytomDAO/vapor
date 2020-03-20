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

type chainData struct {
	bestBlockHeader    *types.BlockHeader
	lastIrrBlockHeader *types.BlockHeader
	utxoViewPoint      *state.UtxoViewpoint
	storedBlocks       []*types.Block
	consensusResults   []*state.ConsensusResult
}

func TestRollback(t *testing.T) {
	cases := []struct {
		desc                   string
		movStartHeight         uint64
		RoundVoteBlockNums     uint64
		beforeChainData        *chainData
		wantChainData          *chainData
		rollbackToTargetHeight uint64
	}{
		{
			desc:                   "rollback from height 1 to 0",
			movStartHeight:         10,
			RoundVoteBlockNums:     1200,
			rollbackToTargetHeight: 0,
			beforeChainData: &chainData{
				bestBlockHeader: &types.BlockHeader{
					Height:            1,
					PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
				},
				lastIrrBlockHeader: &types.BlockHeader{
					Height:            1,
					PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
				},
				utxoViewPoint: &state.UtxoViewpoint{
					Entries: map[bc.Hash]*storage.UtxoEntry{
						testutil.MustDecodeHash("51f538be366172bed5359a016dce26b952024c9607caf6af609ad723982c2e06"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
						testutil.MustDecodeHash("e2370262a129b90174195a76c298d872a56af042eae17657e154bcc46d41b3ba"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
					},
				},
				storedBlocks: []*types.Block{
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
				consensusResults: []*state.ConsensusResult{
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
			},
			wantChainData: &chainData{
				bestBlockHeader: &types.BlockHeader{
					Height: 0,
				},
				lastIrrBlockHeader: &types.BlockHeader{
					Height: 0,
				},
				storedBlocks: []*types.Block{
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
				utxoViewPoint: &state.UtxoViewpoint{
					Entries: map[bc.Hash]*storage.UtxoEntry{
						testutil.MustDecodeHash("e2370262a129b90174195a76c298d872a56af042eae17657e154bcc46d41b3ba"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
					},
				},
				consensusResults: []*state.ConsensusResult{
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
		},
		{
			desc:                   "rollback from height 2 to 0",
			movStartHeight:         10,
			RoundVoteBlockNums:     1200,
			rollbackToTargetHeight: 0,
			beforeChainData: &chainData{
				bestBlockHeader: &types.BlockHeader{
					Height:            2,
					PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
				},
				lastIrrBlockHeader: &types.BlockHeader{
					Height:            2,
					PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
				},
				utxoViewPoint: &state.UtxoViewpoint{
					Entries: map[bc.Hash]*storage.UtxoEntry{
						testutil.MustDecodeHash("afee09925bea1695424450a91ad082a378f20534627fa5cb63f036846347ee08"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 2, Spent: true},
						testutil.MustDecodeHash("51f538be366172bed5359a016dce26b952024c9607caf6af609ad723982c2e06"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
						testutil.MustDecodeHash("e2370262a129b90174195a76c298d872a56af042eae17657e154bcc46d41b3ba"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
					},
				},
				storedBlocks: []*types.Block{
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
				consensusResults: []*state.ConsensusResult{
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
			},
			wantChainData: &chainData{
				bestBlockHeader: &types.BlockHeader{
					Height: 0,
				},
				lastIrrBlockHeader: &types.BlockHeader{
					Height: 0,
				},
				storedBlocks: []*types.Block{
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
				utxoViewPoint: &state.UtxoViewpoint{
					Entries: map[bc.Hash]*storage.UtxoEntry{
						testutil.MustDecodeHash("e2370262a129b90174195a76c298d872a56af042eae17657e154bcc46d41b3ba"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
					},
				},
				consensusResults: []*state.ConsensusResult{
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
		},
		{
			desc:                   "rollback from height 2 to 1",
			movStartHeight:         10,
			RoundVoteBlockNums:     1200,
			rollbackToTargetHeight: 1,
			beforeChainData: &chainData{
				bestBlockHeader: &types.BlockHeader{
					Height:            2,
					PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
				},
				lastIrrBlockHeader: &types.BlockHeader{
					Height:            2,
					PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
				},
				utxoViewPoint: &state.UtxoViewpoint{
					Entries: map[bc.Hash]*storage.UtxoEntry{
						testutil.MustDecodeHash("afee09925bea1695424450a91ad082a378f20534627fa5cb63f036846347ee08"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 2, Spent: true},
						testutil.MustDecodeHash("51f538be366172bed5359a016dce26b952024c9607caf6af609ad723982c2e06"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
						testutil.MustDecodeHash("e2370262a129b90174195a76c298d872a56af042eae17657e154bcc46d41b3ba"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
					},
				},
				storedBlocks: []*types.Block{
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
				consensusResults: []*state.ConsensusResult{
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
			},
			wantChainData: &chainData{
				bestBlockHeader: &types.BlockHeader{
					Height:            1,
					PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
				},
				lastIrrBlockHeader: &types.BlockHeader{
					Height:            1,
					PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
				},
				storedBlocks: []*types.Block{
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
				utxoViewPoint: &state.UtxoViewpoint{
					Entries: map[bc.Hash]*storage.UtxoEntry{
						testutil.MustDecodeHash("51f538be366172bed5359a016dce26b952024c9607caf6af609ad723982c2e06"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
						testutil.MustDecodeHash("e2370262a129b90174195a76c298d872a56af042eae17657e154bcc46d41b3ba"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
					},
				},
				consensusResults: []*state.ConsensusResult{
					{
						Seq: 1,
						NumOfVote: map[string]uint64{
							"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100002000,
							"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 100002000,
						},
						BlockHash:      testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
						BlockHeight:    1,
						CoinbaseReward: map[string]uint64{"0001": consensus.BlockSubsidy(1)},
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
			},
		},
		{
			desc:                   "rollback from height 2 to 1, RoundVoteBlockNums is 2",
			movStartHeight:         10,
			RoundVoteBlockNums:     2,
			rollbackToTargetHeight: 1,
			beforeChainData: &chainData{
				bestBlockHeader: &types.BlockHeader{
					Height:            2,
					PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
				},
				lastIrrBlockHeader: &types.BlockHeader{
					Height:            2,
					PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
				},
				utxoViewPoint: &state.UtxoViewpoint{
					Entries: map[bc.Hash]*storage.UtxoEntry{
						testutil.MustDecodeHash("afee09925bea1695424450a91ad082a378f20534627fa5cb63f036846347ee08"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 2, Spent: true},
						testutil.MustDecodeHash("51f538be366172bed5359a016dce26b952024c9607caf6af609ad723982c2e06"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
						testutil.MustDecodeHash("e2370262a129b90174195a76c298d872a56af042eae17657e154bcc46d41b3ba"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
					},
				},
				storedBlocks: []*types.Block{
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
				consensusResults: []*state.ConsensusResult{
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
			},
			wantChainData: &chainData{
				bestBlockHeader: &types.BlockHeader{
					Height:            1,
					PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
				},
				lastIrrBlockHeader: &types.BlockHeader{
					Height:            1,
					PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
				},
				storedBlocks: []*types.Block{
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
				utxoViewPoint: &state.UtxoViewpoint{
					Entries: map[bc.Hash]*storage.UtxoEntry{
						testutil.MustDecodeHash("51f538be366172bed5359a016dce26b952024c9607caf6af609ad723982c2e06"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
						testutil.MustDecodeHash("e2370262a129b90174195a76c298d872a56af042eae17657e154bcc46d41b3ba"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
					},
				},
				consensusResults: []*state.ConsensusResult{
					{
						Seq: 1,
						NumOfVote: map[string]uint64{
							"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100002000,
							"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 100002000,
						},
						BlockHash:      testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
						BlockHeight:    1,
						CoinbaseReward: map[string]uint64{"0001": consensus.BlockSubsidy(1)},
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
			},
		},
		{
			desc:                   "rollback from height 3 to 1, RoundVoteBlockNums is 2",
			movStartHeight:         10,
			RoundVoteBlockNums:     2,
			rollbackToTargetHeight: 1,
			beforeChainData: &chainData{
				bestBlockHeader: &types.BlockHeader{
					Height:            3,
					PreviousBlockHash: testutil.MustDecodeHash("699d3f59d4afe7eea85df31814628d7d34ace7f5e76d6c9ebf4c54482d2cd333"),
				},
				lastIrrBlockHeader: &types.BlockHeader{
					Height:            3,
					PreviousBlockHash: testutil.MustDecodeHash("699d3f59d4afe7eea85df31814628d7d34ace7f5e76d6c9ebf4c54482d2cd333"),
				},
				utxoViewPoint: &state.UtxoViewpoint{
					Entries: map[bc.Hash]*storage.UtxoEntry{
						testutil.MustDecodeHash("5b4d53fbc2a489847f34dd0e0c085797fe7cf0a3a9a2f3231d11bdad16dea2be"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 3, Spent: true},
						testutil.MustDecodeHash("4c2b719d10fc6b9c2a7c343491ddd8c0d6bd57f9c6680bfda557689c182cf685"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 2, Spent: true},
						testutil.MustDecodeHash("9fb6f213e3130810e755675707d0e9870c79a91c575638a580fae65568ca9e99"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
						testutil.MustDecodeHash("3d1617908e624a2042c23be4f671b261d5b8a2a61b8421ee6a702c6e071428a8"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
					},
				},
				storedBlocks: []*types.Block{
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
				consensusResults: []*state.ConsensusResult{
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
			},
			wantChainData: &chainData{
				bestBlockHeader: &types.BlockHeader{
					Height:            1,
					PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
				},
				lastIrrBlockHeader: &types.BlockHeader{
					Height:            1,
					PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
				},
				storedBlocks: []*types.Block{
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
				utxoViewPoint: &state.UtxoViewpoint{
					Entries: map[bc.Hash]*storage.UtxoEntry{
						testutil.MustDecodeHash("9fb6f213e3130810e755675707d0e9870c79a91c575638a580fae65568ca9e99"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
						testutil.MustDecodeHash("3d1617908e624a2042c23be4f671b261d5b8a2a61b8421ee6a702c6e071428a8"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
					},
				},
				consensusResults: []*state.ConsensusResult{
					{
						Seq: 1,
						NumOfVote: map[string]uint64{
							"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100000000 + 100000000 - 2000,
							"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 200000000,
						},
						BlockHash:      testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
						BlockHeight:    1,
						CoinbaseReward: map[string]uint64{"51": consensus.BlockSubsidy(1) + 2000},
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
			},
		},
		{
			desc:                   "rollback from height 3 to 2, RoundVoteBlockNums is 2",
			movStartHeight:         10,
			RoundVoteBlockNums:     2,
			rollbackToTargetHeight: 2,
			beforeChainData: &chainData{
				bestBlockHeader: &types.BlockHeader{
					Height:            3,
					PreviousBlockHash: testutil.MustDecodeHash("699d3f59d4afe7eea85df31814628d7d34ace7f5e76d6c9ebf4c54482d2cd333"),
				},
				lastIrrBlockHeader: &types.BlockHeader{
					Height:            3,
					PreviousBlockHash: testutil.MustDecodeHash("699d3f59d4afe7eea85df31814628d7d34ace7f5e76d6c9ebf4c54482d2cd333"),
				},
				utxoViewPoint: &state.UtxoViewpoint{
					Entries: map[bc.Hash]*storage.UtxoEntry{
						testutil.MustDecodeHash("5b4d53fbc2a489847f34dd0e0c085797fe7cf0a3a9a2f3231d11bdad16dea2be"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 3, Spent: true},
						testutil.MustDecodeHash("4c2b719d10fc6b9c2a7c343491ddd8c0d6bd57f9c6680bfda557689c182cf685"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 2, Spent: true},
						testutil.MustDecodeHash("9fb6f213e3130810e755675707d0e9870c79a91c575638a580fae65568ca9e99"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
						testutil.MustDecodeHash("3d1617908e624a2042c23be4f671b261d5b8a2a61b8421ee6a702c6e071428a8"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
					},
				},
				storedBlocks: []*types.Block{
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
				consensusResults: []*state.ConsensusResult{
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
			},
			wantChainData: &chainData{
				bestBlockHeader: &types.BlockHeader{
					Height:            2,
					PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
				},
				lastIrrBlockHeader: &types.BlockHeader{
					Height:            2,
					PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
				},
				storedBlocks: []*types.Block{
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
				utxoViewPoint: &state.UtxoViewpoint{
					Entries: map[bc.Hash]*storage.UtxoEntry{
						testutil.MustDecodeHash("9fb6f213e3130810e755675707d0e9870c79a91c575638a580fae65568ca9e99"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
						testutil.MustDecodeHash("3d1617908e624a2042c23be4f671b261d5b8a2a61b8421ee6a702c6e071428a8"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
						testutil.MustDecodeHash("4c2b719d10fc6b9c2a7c343491ddd8c0d6bd57f9c6680bfda557689c182cf685"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 2, Spent: true},
					},
				},
				consensusResults: []*state.ConsensusResult{
					{
						Seq: 1,
						NumOfVote: map[string]uint64{
							"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100000000 + 100000000 - 2000 + 250000000,
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
			},
		},
		{
			desc:                   "rollback from height 4 to 2, there is two chain , and round vote block nums is 2",
			movStartHeight:         10,
			RoundVoteBlockNums:     2,
			rollbackToTargetHeight: 2,
			beforeChainData: &chainData{
				bestBlockHeader: &types.BlockHeader{
					Height:            5,
					Timestamp:         uint64(1528945008),
					PreviousBlockHash: testutil.MustDecodeHash("64a41230412f26a5c0a1734515d9e177bd3573be2ae1d55c4533509a7c9cce8e"),
				},
				lastIrrBlockHeader: &types.BlockHeader{
					Height:            5,
					Timestamp:         uint64(1528945008),
					PreviousBlockHash: testutil.MustDecodeHash("64a41230412f26a5c0a1734515d9e177bd3573be2ae1d55c4533509a7c9cce8e"),
				},
				utxoViewPoint: &state.UtxoViewpoint{
					Entries: map[bc.Hash]*storage.UtxoEntry{
						testutil.MustDecodeHash("3c07f3159d4e2a0527129d644a8fcd09ce26555e94c9c7f348464120ef463275"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 5, Spent: true},
						testutil.MustDecodeHash("927144d2a391e17dc12184f5ae163b994984132ad72c34d854bb9009b68cd4cc"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 4, Spent: true},
						testutil.MustDecodeHash("fa43f4ca43bcb0e94d43b52c56d1740dea1329b59a44f6ee045d70446881c514"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 3, Spent: true},
						testutil.MustDecodeHash("f081ccd0c97ae34bc5580a0405d9b1ed0b0ed9e1410f1786b7112b348a412e3d"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 4, Spent: true},
						testutil.MustDecodeHash("2704fa67c76e020b08ffa3f93a500acebcaf68b45ba43d8b3b08b68c5bb1eff1"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 3, Spent: true},
						testutil.MustDecodeHash("4c2b719d10fc6b9c2a7c343491ddd8c0d6bd57f9c6680bfda557689c182cf685"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 2, Spent: true},
						testutil.MustDecodeHash("9fb6f213e3130810e755675707d0e9870c79a91c575638a580fae65568ca9e99"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
						testutil.MustDecodeHash("3d1617908e624a2042c23be4f671b261d5b8a2a61b8421ee6a702c6e071428a8"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
					},
				},
				storedBlocks: []*types.Block{
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
							Timestamp:         uint64(1528945000),
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
									types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 440000000, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 160000000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
								},
							}),
						},
					},
					{
						BlockHeader: types.BlockHeader{
							Height:            4,
							Timestamp:         uint64(1528945005),
							PreviousBlockHash: testutil.MustDecodeHash("bec3dd0d6fecb80a6f3a0373ec2ae676cc1ce72af83546f3d4672231c9b080e6"),
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
									types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 500000000, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 160000000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
								},
							}),
						},
					},
					{
						BlockHeader: types.BlockHeader{
							Height:            3,
							Timestamp:         uint64(1528945001),
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
									types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 402000000, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 200000000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
								},
							}),
						},
					},
					{
						BlockHeader: types.BlockHeader{
							Height:            4,
							Timestamp:         uint64(1528945006),
							PreviousBlockHash: testutil.MustDecodeHash("1d2d01a97d1239de51b4e7d0fb522f71771d2d4f9a0a559154519859cc44a230"),
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
									types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 410000000, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 170000000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
								},
							}),
						},
					},
					{
						BlockHeader: types.BlockHeader{
							Height:            5,
							Timestamp:         uint64(1528945008),
							PreviousBlockHash: testutil.MustDecodeHash("64a41230412f26a5c0a1734515d9e177bd3573be2ae1d55c4533509a7c9cce8e"),
						},
						Transactions: []*types.Tx{
							types.NewTx(types.TxData{
								Inputs: []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
								Outputs: []*types.TxOutput{
									types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51}),
									types.NewIntraChainOutput(bc.AssetID{}, consensus.BlockSubsidy(3)+consensus.BlockSubsidy(4)+520000000, []byte{0x51}),
								},
							}),
							types.NewTx(types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{8}), *consensus.BTMAssetID, 400004000, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 160004000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
								},
							}),
						},
					},
				},
				consensusResults: []*state.ConsensusResult{
					{
						Seq: 3,
						NumOfVote: map[string]uint64{
							"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 980002000,
							"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 200000000,
						},
						BlockHash:      testutil.MustDecodeHash("075ce54f7d4c1b524474265219be52238beec98138f0c0a4d21f1a6b0047914a"),
						BlockHeight:    5,
						CoinbaseReward: map[string]uint64{"51": consensus.BlockSubsidy(5) + 240000000},
					},
					{
						Seq: 2,
						NumOfVote: map[string]uint64{
							"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 819998000,
							"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 200000000,
						},
						BlockHash:      testutil.MustDecodeHash("64a41230412f26a5c0a1734515d9e177bd3573be2ae1d55c4533509a7c9cce8e"),
						BlockHeight:    4,
						CoinbaseReward: map[string]uint64{"51": consensus.BlockSubsidy(3) + consensus.BlockSubsidy(4) + 442000000},
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
			},
			wantChainData: &chainData{
				bestBlockHeader: &types.BlockHeader{
					Height:            2,
					PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
				},
				lastIrrBlockHeader: &types.BlockHeader{
					Height:            2,
					PreviousBlockHash: testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
				},
				storedBlocks: []*types.Block{
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
				consensusResults: []*state.ConsensusResult{
					{
						Seq: 1,
						NumOfVote: map[string]uint64{
							"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100000000 + 100000000 - 2000 + 250000000,
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
				utxoViewPoint: &state.UtxoViewpoint{
					Entries: map[bc.Hash]*storage.UtxoEntry{
						testutil.MustDecodeHash("9fb6f213e3130810e755675707d0e9870c79a91c575638a580fae65568ca9e99"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 1, Spent: true},
						testutil.MustDecodeHash("3d1617908e624a2042c23be4f671b261d5b8a2a61b8421ee6a702c6e071428a8"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 0, Spent: true},
						testutil.MustDecodeHash("4c2b719d10fc6b9c2a7c343491ddd8c0d6bd57f9c6680bfda557689c182cf685"): &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 2, Spent: true},
					},
				},
			},
		},
	}

	for i, c := range cases {
		consensus.ActiveNetParams.RoundVoteBlockNums = c.RoundVoteBlockNums

		movDB := dbm.NewDB("mov_db", "leveldb", "mov_db")
		movCore := mov.NewCoreWithDB(movDatabase.NewLevelDBMovStore(movDB), c.movStartHeight)

		blockDB := dbm.NewDB("block_db", "leveldb", "block_db")
		store := database.NewStore(blockDB)

		mustSaveBlocks(c.beforeChainData.storedBlocks, store)

		var mainChainBlockHeaders []*types.BlockHeader
		for _, block := range c.beforeChainData.storedBlocks {
			mainChainBlockHeaders = append(mainChainBlockHeaders, &block.BlockHeader)
		}
		if err := store.SaveChainStatus(c.beforeChainData.bestBlockHeader, c.beforeChainData.lastIrrBlockHeader, mainChainBlockHeaders, c.beforeChainData.utxoViewPoint, c.beforeChainData.consensusResults); err != nil {
			t.Fatal(err)
		}

		chain, err := protocol.NewChain(store, nil, []protocol.Protocoler{movCore}, nil)
		if err != nil {
			t.Fatal(err)
		}

		if err := chain.Rollback(c.rollbackToTargetHeight); err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(chain.LastIrreversibleHeader(), c.wantChainData.lastIrrBlockHeader) {
			t.Errorf("lastIrrBlockHeader is not right!")
		}

		if !testutil.DeepEqual(chain.BestBlockHeader(), c.wantChainData.bestBlockHeader) {
			t.Errorf("wantBestBlockHeader is not right!")
		}

		gotConsensusResults := mustGetConsensusResultFromStore(store, chain)
		if !testutil.DeepEqual(gotConsensusResults, c.wantChainData.consensusResults) {
			t.Errorf("cases#%d(%s) wantBestConsensusResult is not right!", i, c.desc)
		}

		gotBlocks := mustGetBlocksFromStore(chain)
		if !blocksEquals(gotBlocks, c.wantChainData.storedBlocks) {
			t.Errorf("cases#%d(%s) the blocks is not same!", i, c.desc)
		}

		gotTransactions := getBcTransactions(gotBlocks)
		gotUtxoViewPoint := state.NewUtxoViewpoint()
		if err = store.GetTransactionsUtxo(gotUtxoViewPoint, gotTransactions); err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(gotUtxoViewPoint, c.wantChainData.utxoViewPoint) {
			t.Fatal(err)
		}

		blockDB.Close()
		os.RemoveAll("block_db")
		movDB.Close()
		os.RemoveAll("mov_db")

	}
}

func blocksEquals(blocks1 []*types.Block, blocks2 []*types.Block) bool {
	blockHashMap1 := make(map[string]interface{})
	for _, block := range blocks1 {
		hash := block.Hash()
		blockHashMap1[hash.String()] = nil
	}

	blockHashMap2 := make(map[string]interface{})
	for _, block := range blocks2 {
		hash := block.Hash()
		blockHashMap2[hash.String()] = nil
	}
	return testutil.DeepEqual(blockHashMap1, blockHashMap2)
}

func getBcTransactions(blocks []*types.Block) []*bc.Tx {
	var txs []*bc.Tx
	for _, block := range blocks {
		for _, tx := range block.Transactions {
			txs = append(txs, tx.Tx)
		}
	}
	return txs
}

func mustSaveBlocks(blocks []*types.Block, store *database.Store) {
	for _, block := range blocks {
		status := bc.NewTransactionStatus()
		for index := range block.Transactions {
			if err := status.SetStatus(index, false); err != nil {
				panic(err)
			}
		}
		if err := store.SaveBlock(block, status); err != nil {
			panic(err)
		}
	}
}

func mustGetBlocksFromStore(chain *protocol.Chain) []*types.Block {
	var blocks []*types.Block
	for height := int64(chain.BestBlockHeight()); height >= 0; height-- {
		block, err := chain.GetBlockByHeight(uint64(height))
		if err != nil {
			panic(err)
		}

		blocks = append(blocks, block)
	}
	return blocks
}

func mustGetConsensusResultFromStore(store *database.Store, chain *protocol.Chain) []*state.ConsensusResult {
	var consensusResults []*state.ConsensusResult
	for seq := int64(state.CalcVoteSeq(chain.BestBlockHeight())); seq >= 0; seq-- {
		consensusResult, err := store.GetConsensusResult(uint64(seq))
		if err != nil {
			panic(err)
		}

		consensusResults = append(consensusResults, consensusResult)
	}
	return consensusResults
}
