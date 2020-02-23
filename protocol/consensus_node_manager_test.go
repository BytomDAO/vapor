package protocol

import (
	"testing"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/crypto/ed25519/chainkd"
	"github.com/bytom/vapor/database/storage"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
	"github.com/bytom/vapor/testutil"
)

func TestGetConsensusNodes(t *testing.T) {
	cases := []struct {
		desc                   string
		prevSeqConsensusResult *state.ConsensusResult
		storedBlockHeaders     []*types.BlockHeader
		storedBlocks           []*types.Block
		prevBlockHash          bc.Hash
		bestBlockHeight        uint64
		wantConsensusNodes     []*state.ConsensusNode
	}{
		{
			desc: "block hash in main chain",
			prevSeqConsensusResult: &state.ConsensusResult{
				Seq: 1,
				NumOfVote: map[string]uint64{
					"0f8669abbd3cc0a167156188e428f940088d5b2f36bb3449df71d2bdc5e077814ea3f68628eef279ed435f51ee26cff00f8bd28fabfd500bedb2a9e369f5c825": 838063475500000,
					"e7f458ee8d2ba19b0fdc7410d1fd57e9c2e1a79377c661d66c55effe49d7ffc920e40510442d4a10b7bea06c09fb0b41f52601135adaaa7136204db36106c093": 474794800000000,
					"1bec3a35da038ec7a76c40986e80b5af2dcef60341970e3fc58b4db0797bd4ca9b2cbf3d7ab820832e22a80b5b86ae1427f7f706a7780089958b2862e7bc0842": 833812985000000,
					"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 285918061999999,
					"b928e46bb01e834fdf167185e31b15de7cc257af8bbdf17f9c7fefd5bb97b306d048b6bc0da2097152c1c2ff38333c756a543adbba7030a447dcc776b8ac64ef": 1228455289930297,
					"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 274387690000000,
				},
				BlockHash:   testutil.MustDecodeHash("65a92adc3c17657e4f0f032f1de83597dbbc6c29c24626d1533c8215f81c99e6"),
				BlockHeight: 1200,
			},
			storedBlockHeaders: []*types.BlockHeader{
				{
					Height:            1200,
					PreviousBlockHash: testutil.MustDecodeHash("7f5d567f2ad9de9af4af6e6cc81943abcd042e5a6329c83708e3578be3b5fc7d"),
				},
				{
					Height:            1201,
					PreviousBlockHash: testutil.MustDecodeHash("65a92adc3c17657e4f0f032f1de83597dbbc6c29c24626d1533c8215f81c99e6"),
				},
				{
					Height:            1202,
					PreviousBlockHash: testutil.MustDecodeHash("9ff85d12c91ad5500b3dfe80bb0b4d4496ab29aec37f71e839fdd0a1c92a614f"),
				},
				{
					Height:            1203,
					PreviousBlockHash: testutil.MustDecodeHash("559b02b183e275e79dcfab23c66004264127f461ff8b90b3a336b2f958bf96e2"),
				},
				{
					Height:            1204,
					PreviousBlockHash: testutil.MustDecodeHash("f77d1330abc1a58ce5909581e6f2059153e116e9044e284609e07efd5f0b239e"),
				},
			},
			prevBlockHash:   testutil.MustDecodeHash("f77d1330abc1a58ce5909581e6f2059153e116e9044e284609e07efd5f0b239e"), // 1204
			bestBlockHeight: 1230,
			wantConsensusNodes: []*state.ConsensusNode{
				{
					XPub:    mustDecodeXPub("b928e46bb01e834fdf167185e31b15de7cc257af8bbdf17f9c7fefd5bb97b306d048b6bc0da2097152c1c2ff38333c756a543adbba7030a447dcc776b8ac64ef"),
					VoteNum: 1228455289930297,
					Order:   0,
				},
				{
					XPub:    mustDecodeXPub("0f8669abbd3cc0a167156188e428f940088d5b2f36bb3449df71d2bdc5e077814ea3f68628eef279ed435f51ee26cff00f8bd28fabfd500bedb2a9e369f5c825"),
					VoteNum: 838063475500000,
					Order:   1,
				},
				{
					XPub:    mustDecodeXPub("1bec3a35da038ec7a76c40986e80b5af2dcef60341970e3fc58b4db0797bd4ca9b2cbf3d7ab820832e22a80b5b86ae1427f7f706a7780089958b2862e7bc0842"),
					VoteNum: 833812985000000,
					Order:   2,
				},
				{
					XPub:    mustDecodeXPub("e7f458ee8d2ba19b0fdc7410d1fd57e9c2e1a79377c661d66c55effe49d7ffc920e40510442d4a10b7bea06c09fb0b41f52601135adaaa7136204db36106c093"),
					VoteNum: 474794800000000,
					Order:   3,
				},
				{
					XPub:    mustDecodeXPub("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9"),
					VoteNum: 285918061999999,
					Order:   4,
				},
				{
					XPub:    mustDecodeXPub("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67"),
					VoteNum: 274387690000000,
					Order:   5,
				},
			},
		},
		{
			desc: "block hash in main chain, block height in the begin of round",
			prevSeqConsensusResult: &state.ConsensusResult{
				Seq: 1,
				NumOfVote: map[string]uint64{
					"0f8669abbd3cc0a167156188e428f940088d5b2f36bb3449df71d2bdc5e077814ea3f68628eef279ed435f51ee26cff00f8bd28fabfd500bedb2a9e369f5c825": 838063475500000,
					"e7f458ee8d2ba19b0fdc7410d1fd57e9c2e1a79377c661d66c55effe49d7ffc920e40510442d4a10b7bea06c09fb0b41f52601135adaaa7136204db36106c093": 474794800000000,
					"1bec3a35da038ec7a76c40986e80b5af2dcef60341970e3fc58b4db0797bd4ca9b2cbf3d7ab820832e22a80b5b86ae1427f7f706a7780089958b2862e7bc0842": 833812985000000,
					"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 285918061999999,
					"b928e46bb01e834fdf167185e31b15de7cc257af8bbdf17f9c7fefd5bb97b306d048b6bc0da2097152c1c2ff38333c756a543adbba7030a447dcc776b8ac64ef": 1228455289930297,
					"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 274387690000000,
				},
				BlockHash:   testutil.MustDecodeHash("65a92adc3c17657e4f0f032f1de83597dbbc6c29c24626d1533c8215f81c99e6"),
				BlockHeight: 1200,
			},
			storedBlockHeaders: []*types.BlockHeader{
				{
					Height:            1200,
					PreviousBlockHash: testutil.MustDecodeHash("7f5d567f2ad9de9af4af6e6cc81943abcd042e5a6329c83708e3578be3b5fc7d"),
				},
				{
					Height:            1201,
					PreviousBlockHash: testutil.MustDecodeHash("65a92adc3c17657e4f0f032f1de83597dbbc6c29c24626d1533c8215f81c99e6"),
				},
			},
			prevBlockHash:   testutil.MustDecodeHash("65a92adc3c17657e4f0f032f1de83597dbbc6c29c24626d1533c8215f81c99e6"), // 1201
			bestBlockHeight: 1230,
			wantConsensusNodes: []*state.ConsensusNode{
				{
					XPub:    mustDecodeXPub("b928e46bb01e834fdf167185e31b15de7cc257af8bbdf17f9c7fefd5bb97b306d048b6bc0da2097152c1c2ff38333c756a543adbba7030a447dcc776b8ac64ef"),
					VoteNum: 1228455289930297,
					Order:   0,
				},
				{
					XPub:    mustDecodeXPub("0f8669abbd3cc0a167156188e428f940088d5b2f36bb3449df71d2bdc5e077814ea3f68628eef279ed435f51ee26cff00f8bd28fabfd500bedb2a9e369f5c825"),
					VoteNum: 838063475500000,
					Order:   1,
				},
				{
					XPub:    mustDecodeXPub("1bec3a35da038ec7a76c40986e80b5af2dcef60341970e3fc58b4db0797bd4ca9b2cbf3d7ab820832e22a80b5b86ae1427f7f706a7780089958b2862e7bc0842"),
					VoteNum: 833812985000000,
					Order:   2,
				},
				{
					XPub:    mustDecodeXPub("e7f458ee8d2ba19b0fdc7410d1fd57e9c2e1a79377c661d66c55effe49d7ffc920e40510442d4a10b7bea06c09fb0b41f52601135adaaa7136204db36106c093"),
					VoteNum: 474794800000000,
					Order:   3,
				},
				{
					XPub:    mustDecodeXPub("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9"),
					VoteNum: 285918061999999,
					Order:   4,
				},
				{
					XPub:    mustDecodeXPub("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67"),
					VoteNum: 274387690000000,
					Order:   5,
				},
			},
		},
		{
			desc: "block hash in main chain, the consensus result is incomplete",
			prevSeqConsensusResult: &state.ConsensusResult{
				Seq: 1,
				NumOfVote: map[string]uint64{
					"0f8669abbd3cc0a167156188e428f940088d5b2f36bb3449df71d2bdc5e077814ea3f68628eef279ed435f51ee26cff00f8bd28fabfd500bedb2a9e369f5c825": 838063475500000,
					"e7f458ee8d2ba19b0fdc7410d1fd57e9c2e1a79377c661d66c55effe49d7ffc920e40510442d4a10b7bea06c09fb0b41f52601135adaaa7136204db36106c093": 474794800000000,
					"1bec3a35da038ec7a76c40986e80b5af2dcef60341970e3fc58b4db0797bd4ca9b2cbf3d7ab820832e22a80b5b86ae1427f7f706a7780089958b2862e7bc0842": 833812985000000,
					"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 285918061999999,
					"b928e46bb01e834fdf167185e31b15de7cc257af8bbdf17f9c7fefd5bb97b306d048b6bc0da2097152c1c2ff38333c756a543adbba7030a447dcc776b8ac64ef": 1228455289930297,
					"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 274387690000000,
				},
				BlockHash:      testutil.MustDecodeHash("ef24de31371b4d34363011b6c8b065b1acaad9264d9abae2253d584e0d3a8739"),
				BlockHeight:    1198,
				CoinbaseReward: make(map[string]uint64),
			},
			storedBlockHeaders: []*types.BlockHeader{
				{
					Height:            1198,
					PreviousBlockHash: testutil.MustDecodeHash("4d3ecf28f3045df23764792f41af17fc20518cc8e83673dc2d0ce07bbf084d7d"),
				},
				{
					Height:            1199,
					PreviousBlockHash: testutil.MustDecodeHash("ef24de31371b4d34363011b6c8b065b1acaad9264d9abae2253d584e0d3a8739"),
				},
				{
					Height:            1200,
					PreviousBlockHash: testutil.MustDecodeHash("608d82b3660255186fb2b035cfb2ca86e986433218cd2eca2e79611e855a87d3"),
				},
				{
					Height:            1201,
					PreviousBlockHash: testutil.MustDecodeHash("73a2a6a098727877e288a4520f2d8076d700b561277ed7c3f533e3f176496888"),
				},
			},
			storedBlocks: []*types.Block{
				{
					BlockHeader: types.BlockHeader{
						Height:            1199,
						PreviousBlockHash: testutil.MustDecodeHash("ef24de31371b4d34363011b6c8b065b1acaad9264d9abae2253d584e0d3a8739"),
					},
					Transactions: []*types.Tx{
						{
							TxData: types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 1E14, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 1E14, []byte{0, 1}, testutil.MustDecodeHexString("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67")),
								},
							},
						},
					},
				},
				{
					BlockHeader: types.BlockHeader{
						Height:            1200,
						PreviousBlockHash: testutil.MustDecodeHash("608d82b3660255186fb2b035cfb2ca86e986433218cd2eca2e79611e855a87d3"),
					},
					Transactions: []*types.Tx{
						{
							TxData: types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 3E14, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 3E14, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
								},
							},
						},
					},
				},
			},
			prevBlockHash:   testutil.MustDecodeHash("73a2a6a098727877e288a4520f2d8076d700b561277ed7c3f533e3f176496888"), // 1201
			bestBlockHeight: 1230,
			wantConsensusNodes: []*state.ConsensusNode{
				{
					XPub:    mustDecodeXPub("b928e46bb01e834fdf167185e31b15de7cc257af8bbdf17f9c7fefd5bb97b306d048b6bc0da2097152c1c2ff38333c756a543adbba7030a447dcc776b8ac64ef"),
					VoteNum: 1228455289930297,
					Order:   0,
				},
				{
					XPub:    mustDecodeXPub("0f8669abbd3cc0a167156188e428f940088d5b2f36bb3449df71d2bdc5e077814ea3f68628eef279ed435f51ee26cff00f8bd28fabfd500bedb2a9e369f5c825"),
					VoteNum: 838063475500000,
					Order:   1,
				},
				{
					XPub:    mustDecodeXPub("1bec3a35da038ec7a76c40986e80b5af2dcef60341970e3fc58b4db0797bd4ca9b2cbf3d7ab820832e22a80b5b86ae1427f7f706a7780089958b2862e7bc0842"),
					VoteNum: 833812985000000,
					Order:   2,
				},
				{
					XPub:    mustDecodeXPub("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9"),
					VoteNum: 585918061999999,
					Order:   3,
				},
				{
					XPub:    mustDecodeXPub("e7f458ee8d2ba19b0fdc7410d1fd57e9c2e1a79377c661d66c55effe49d7ffc920e40510442d4a10b7bea06c09fb0b41f52601135adaaa7136204db36106c093"),
					VoteNum: 474794800000000,
					Order:   4,
				},
				{
					XPub:    mustDecodeXPub("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67"),
					VoteNum: 374387690000000,
					Order:   5,
				},
			},
		},
		{
			desc: "block hash in fork chain",
			prevSeqConsensusResult: &state.ConsensusResult{
				Seq: 1,
				NumOfVote: map[string]uint64{
					"0f8669abbd3cc0a167156188e428f940088d5b2f36bb3449df71d2bdc5e077814ea3f68628eef279ed435f51ee26cff00f8bd28fabfd500bedb2a9e369f5c825": 838063475500000,
					"e7f458ee8d2ba19b0fdc7410d1fd57e9c2e1a79377c661d66c55effe49d7ffc920e40510442d4a10b7bea06c09fb0b41f52601135adaaa7136204db36106c093": 474794800000000,
					"1bec3a35da038ec7a76c40986e80b5af2dcef60341970e3fc58b4db0797bd4ca9b2cbf3d7ab820832e22a80b5b86ae1427f7f706a7780089958b2862e7bc0842": 833812985000000,
					"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 285918061999999,
					"b928e46bb01e834fdf167185e31b15de7cc257af8bbdf17f9c7fefd5bb97b306d048b6bc0da2097152c1c2ff38333c756a543adbba7030a447dcc776b8ac64ef": 1228455289930297,
					"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 274387690000000,
				},
				BlockHash:      testutil.MustDecodeHash("73a2a6a098727877e288a4520f2d8076d700b561277ed7c3f533e3f176496888"),
				BlockHeight:    1200,
				CoinbaseReward: map[string]uint64{"0001": 1E8},
			},
			storedBlockHeaders: []*types.BlockHeader{
				// main chain
				{
					Height:            1198,
					PreviousBlockHash: testutil.MustDecodeHash("4d3ecf28f3045df23764792f41af17fc20518cc8e83673dc2d0ce07bbf084d7d"),
				},
				{
					Height:            1199,
					PreviousBlockHash: testutil.MustDecodeHash("ef24de31371b4d34363011b6c8b065b1acaad9264d9abae2253d584e0d3a8739"),
				},
				{
					Height:            1200,
					PreviousBlockHash: testutil.MustDecodeHash("608d82b3660255186fb2b035cfb2ca86e986433218cd2eca2e79611e855a87d3"),
				},
				{
					Height:            1201,
					PreviousBlockHash: testutil.MustDecodeHash("73a2a6a098727877e288a4520f2d8076d700b561277ed7c3f533e3f176496888"),
				},
				{
					Height:            1202,
					PreviousBlockHash: testutil.MustDecodeHash("a5be1d1177eb027327baedb869f902f74850476d0b9432a30391a3165d3af7cc"),
				},
				// fork chain, fork height in 1198, rollback 1200, 1199, append 1199, 1200
				{
					Height:            1199,
					PreviousBlockHash: testutil.MustDecodeHash("ef24de31371b4d34363011b6c8b065b1acaad9264d9abae2253d584e0d3a8739"),
					Timestamp:         1, // in order to make different hash
				},
				{
					Height:            1200,
					PreviousBlockHash: testutil.MustDecodeHash("cde3d8a99dee1cd44fb37f499c1980338a49ac1b9d5e6f693bd0a6f87c74e4d2"),
				},
				{
					Height:            1201,
					PreviousBlockHash: testutil.MustDecodeHash("f819efb4baecbeb63aff6d4b14b549f1bd35955d461abaa0058a26f74c4f62da"),
				},
			},
			storedBlocks: []*types.Block{
				// main chain
				{
					BlockHeader: types.BlockHeader{
						Height:            1199,
						PreviousBlockHash: testutil.MustDecodeHash("ef24de31371b4d34363011b6c8b065b1acaad9264d9abae2253d584e0d3a8739"),
					},
					Transactions: []*types.Tx{
						{
							TxData: types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 1E14, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 1E14, []byte{0, 1}, testutil.MustDecodeHexString("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67")),
								},
							},
						},
					},
				},
				{
					BlockHeader: types.BlockHeader{
						Height:            1200,
						PreviousBlockHash: testutil.MustDecodeHash("608d82b3660255186fb2b035cfb2ca86e986433218cd2eca2e79611e855a87d3"),
					},
					Transactions: []*types.Tx{
						{
							TxData: types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 2E14, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 2E14, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
								},
							},
						},
					},
				},
				// fork chain
				{
					BlockHeader: types.BlockHeader{
						Height:            1199,
						PreviousBlockHash: testutil.MustDecodeHash("ef24de31371b4d34363011b6c8b065b1acaad9264d9abae2253d584e0d3a8739"),
						Timestamp:         1, // in order to make different hash
					},
					Transactions: []*types.Tx{
						{
							TxData: types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 2E14, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 2E14, []byte{0, 1}, testutil.MustDecodeHexString("0f8669abbd3cc0a167156188e428f940088d5b2f36bb3449df71d2bdc5e077814ea3f68628eef279ed435f51ee26cff00f8bd28fabfd500bedb2a9e369f5c825")),
								},
							},
						},
					},
				},
				{
					BlockHeader: types.BlockHeader{
						Height:            1200,
						PreviousBlockHash: testutil.MustDecodeHash("cde3d8a99dee1cd44fb37f499c1980338a49ac1b9d5e6f693bd0a6f87c74e4d2"),
					},
					Transactions: []*types.Tx{
						{
							TxData: types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 5E14, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 5E14, []byte{0, 1}, testutil.MustDecodeHexString("b928e46bb01e834fdf167185e31b15de7cc257af8bbdf17f9c7fefd5bb97b306d048b6bc0da2097152c1c2ff38333c756a543adbba7030a447dcc776b8ac64ef")),
								},
							},
						},
					},
				},
			},
			prevBlockHash:   testutil.MustDecodeHash("f819efb4baecbeb63aff6d4b14b549f1bd35955d461abaa0058a26f74c4f62da"), // 1201
			bestBlockHeight: 1230,
			wantConsensusNodes: []*state.ConsensusNode{
				{
					XPub:    mustDecodeXPub("b928e46bb01e834fdf167185e31b15de7cc257af8bbdf17f9c7fefd5bb97b306d048b6bc0da2097152c1c2ff38333c756a543adbba7030a447dcc776b8ac64ef"),
					VoteNum: 1728455289930297,
					Order:   0,
				},
				{
					XPub:    mustDecodeXPub("0f8669abbd3cc0a167156188e428f940088d5b2f36bb3449df71d2bdc5e077814ea3f68628eef279ed435f51ee26cff00f8bd28fabfd500bedb2a9e369f5c825"),
					VoteNum: 1038063475500000,
					Order:   1,
				},
				{
					XPub:    mustDecodeXPub("1bec3a35da038ec7a76c40986e80b5af2dcef60341970e3fc58b4db0797bd4ca9b2cbf3d7ab820832e22a80b5b86ae1427f7f706a7780089958b2862e7bc0842"),
					VoteNum: 833812985000000,
					Order:   2,
				},
				{
					XPub:    mustDecodeXPub("e7f458ee8d2ba19b0fdc7410d1fd57e9c2e1a79377c661d66c55effe49d7ffc920e40510442d4a10b7bea06c09fb0b41f52601135adaaa7136204db36106c093"),
					VoteNum: 474794800000000,
					Order:   3,
				},
				{
					XPub:    mustDecodeXPub("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67"),
					VoteNum: 174387690000000,
					Order:   4,
				},
			},
		},
		{
			desc: "block hash in fork chain, the consensus result is incomplete",
			prevSeqConsensusResult: &state.ConsensusResult{
				Seq: 1,
				NumOfVote: map[string]uint64{
					"0f8669abbd3cc0a167156188e428f940088d5b2f36bb3449df71d2bdc5e077814ea3f68628eef279ed435f51ee26cff00f8bd28fabfd500bedb2a9e369f5c825": 838063475500000,
					"e7f458ee8d2ba19b0fdc7410d1fd57e9c2e1a79377c661d66c55effe49d7ffc920e40510442d4a10b7bea06c09fb0b41f52601135adaaa7136204db36106c093": 474794800000000,
					"1bec3a35da038ec7a76c40986e80b5af2dcef60341970e3fc58b4db0797bd4ca9b2cbf3d7ab820832e22a80b5b86ae1427f7f706a7780089958b2862e7bc0842": 833812985000000,
					"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 285918061999999,
					"b928e46bb01e834fdf167185e31b15de7cc257af8bbdf17f9c7fefd5bb97b306d048b6bc0da2097152c1c2ff38333c756a543adbba7030a447dcc776b8ac64ef": 1228455289930297,
					"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 274387690000000,
				},
				BlockHash:      testutil.MustDecodeHash("a0239137437634c933fa1200f801783a532b88e9556a0746b75a5832aac09bfc"),
				BlockHeight:    1198,
				CoinbaseReward: map[string]uint64{"0001": 1E8},
			},
			storedBlockHeaders: []*types.BlockHeader{
				// main chain
				{
					Height:            1197,
					PreviousBlockHash: testutil.MustDecodeHash("dda6bc15d7de8dadbbb283e90d24bd05a9fd0f721023f53822209281c5ce1698"),
				},
				{
					Height:            1198,
					PreviousBlockHash: testutil.MustDecodeHash("b9c3d0ce5eceac94b53208360c7e100bb342a00ba70c990dc48ce959295793e6"),
				},
				{
					Height:            1199,
					PreviousBlockHash: testutil.MustDecodeHash("a0239137437634c933fa1200f801783a532b88e9556a0746b75a5832aac09bfc"),
				},
				{
					Height:            1200,
					PreviousBlockHash: testutil.MustDecodeHash("9e135b0372e670d02fb4a1d4af6b1a4cc1f33fcec3499a2e735f128b6b9ec144"),
				},
				{
					Height:            1201,
					PreviousBlockHash: testutil.MustDecodeHash("c907b3e07505e0da5cbd98e8fb45f5b5a1e3645813fb855555d085c1e104cd5b"),
				},
				{
					Height:            1202,
					PreviousBlockHash: testutil.MustDecodeHash("99fa16d4e2a01093af987ef39080eb76ed8ded7ce22d50d76b4c3e2ac660214a"),
				},
				// fork chain, fork height in 1197, roll back 1198, append 1198, 1199, 1200
				{
					Height:            1198,
					PreviousBlockHash: testutil.MustDecodeHash("b9c3d0ce5eceac94b53208360c7e100bb342a00ba70c990dc48ce959295793e6"),
					Timestamp:         1, // in order to make different hash
				},
				{
					Height:            1199,
					PreviousBlockHash: testutil.MustDecodeHash("dc209b2c1ead7794f81ac9cdafc136875ec9d3b8525f1482a430ce762685aece"),
				},
				{
					Height:            1200,
					PreviousBlockHash: testutil.MustDecodeHash("fe85f3a22c4c41b03904f19475ec265138aae57be05b0c3ade0be2773fd35eb2"),
				},
				{
					Height:            1201,
					PreviousBlockHash: testutil.MustDecodeHash("01f1a4eb35af3347080497c014c3b980bcfc1c836fdbe6b95f6fa0984ac14f8a"),
				},
			},
			storedBlocks: []*types.Block{
				// main chain
				{
					BlockHeader: types.BlockHeader{
						Height:            1197,
						PreviousBlockHash: testutil.MustDecodeHash("dda6bc15d7de8dadbbb283e90d24bd05a9fd0f721023f53822209281c5ce1698"),
					},
				},
				{
					BlockHeader: types.BlockHeader{
						Height:            1198,
						PreviousBlockHash: testutil.MustDecodeHash("b9c3d0ce5eceac94b53208360c7e100bb342a00ba70c990dc48ce959295793e6"),
					},
					Transactions: []*types.Tx{
						{
							TxData: types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 1E14, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 1E14, []byte{0, 1}, testutil.MustDecodeHexString("1bec3a35da038ec7a76c40986e80b5af2dcef60341970e3fc58b4db0797bd4ca9b2cbf3d7ab820832e22a80b5b86ae1427f7f706a7780089958b2862e7bc0842")),
								},
							},
						},
					},
				},
				{
					BlockHeader: types.BlockHeader{
						Height:            1199,
						PreviousBlockHash: testutil.MustDecodeHash("a0239137437634c933fa1200f801783a532b88e9556a0746b75a5832aac09bfc"),
					},
					Transactions: []*types.Tx{
						{
							TxData: types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 1E14, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 1E14, []byte{0, 1}, testutil.MustDecodeHexString("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67")),
								},
							},
						},
					},
				},
				{
					BlockHeader: types.BlockHeader{
						Height:            1200,
						PreviousBlockHash: testutil.MustDecodeHash("9e135b0372e670d02fb4a1d4af6b1a4cc1f33fcec3499a2e735f128b6b9ec144"),
					},
					Transactions: []*types.Tx{
						{
							TxData: types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 2E14, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 2E14, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
								},
							},
						},
					},
				},
				// fork chain
				{
					BlockHeader: types.BlockHeader{
						Height:            1198,
						PreviousBlockHash: testutil.MustDecodeHash("b9c3d0ce5eceac94b53208360c7e100bb342a00ba70c990dc48ce959295793e6"),
						Timestamp:         1, // in order to make different hash
					},
					Transactions: []*types.Tx{
						{
							TxData: types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 5E14, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 5E14, []byte{0, 1}, testutil.MustDecodeHexString("1bec3a35da038ec7a76c40986e80b5af2dcef60341970e3fc58b4db0797bd4ca9b2cbf3d7ab820832e22a80b5b86ae1427f7f706a7780089958b2862e7bc0842")),
								},
							},
						},
					},
				},
				{
					BlockHeader: types.BlockHeader{
						Height:            1199,
						PreviousBlockHash: testutil.MustDecodeHash("dc209b2c1ead7794f81ac9cdafc136875ec9d3b8525f1482a430ce762685aece"),
					},
					Transactions: []*types.Tx{
						{
							TxData: types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 3E14, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 3E14, []byte{0, 1}, testutil.MustDecodeHexString("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67")),
								},
							},
						},
					},
				},
				{
					BlockHeader: types.BlockHeader{
						Height:            1200,
						PreviousBlockHash: testutil.MustDecodeHash("fe85f3a22c4c41b03904f19475ec265138aae57be05b0c3ade0be2773fd35eb2"),
					},
					Transactions: []*types.Tx{
						{
							TxData: types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 5E14, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 5E14, []byte{0, 1}, testutil.MustDecodeHexString("b928e46bb01e834fdf167185e31b15de7cc257af8bbdf17f9c7fefd5bb97b306d048b6bc0da2097152c1c2ff38333c756a543adbba7030a447dcc776b8ac64ef")),
								},
							},
						},
					},
				},
			},
			prevBlockHash:   testutil.MustDecodeHash("01f1a4eb35af3347080497c014c3b980bcfc1c836fdbe6b95f6fa0984ac14f8a"), // 1201
			bestBlockHeight: 1230,
			wantConsensusNodes: []*state.ConsensusNode{
				{
					XPub:    mustDecodeXPub("b928e46bb01e834fdf167185e31b15de7cc257af8bbdf17f9c7fefd5bb97b306d048b6bc0da2097152c1c2ff38333c756a543adbba7030a447dcc776b8ac64ef"),
					VoteNum: 1728455289930297,
					Order:   0,
				},
				{
					XPub:    mustDecodeXPub("1bec3a35da038ec7a76c40986e80b5af2dcef60341970e3fc58b4db0797bd4ca9b2cbf3d7ab820832e22a80b5b86ae1427f7f706a7780089958b2862e7bc0842"),
					VoteNum: 1233812985000000,
					Order:   1,
				},
				{
					XPub:    mustDecodeXPub("0f8669abbd3cc0a167156188e428f940088d5b2f36bb3449df71d2bdc5e077814ea3f68628eef279ed435f51ee26cff00f8bd28fabfd500bedb2a9e369f5c825"),
					VoteNum: 838063475500000,
					Order:   2,
				},
				{
					XPub:    mustDecodeXPub("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67"),
					VoteNum: 574387690000000,
					Order:   3,
				},
				{
					XPub:    mustDecodeXPub("e7f458ee8d2ba19b0fdc7410d1fd57e9c2e1a79377c661d66c55effe49d7ffc920e40510442d4a10b7bea06c09fb0b41f52601135adaaa7136204db36106c093"),
					VoteNum: 474794800000000,
					Order:   4,
				},
				{
					XPub:    mustDecodeXPub("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9"),
					VoteNum: 285918061999999,
					Order:   5,
				},
			},
		},
	}

	for i, c := range cases {
		store := newDummyStore()
		for _, header := range c.storedBlockHeaders {
			store.SaveBlockHeader(header)
		}

		for _, block := range c.storedBlocks {
			store.SaveBlock(block, nil)
		}

		store.SetConsensusResult(c.prevSeqConsensusResult)

		chain := &Chain{
			store:           store,
			bestBlockHeader: &types.BlockHeader{Height: c.bestBlockHeight},
		}
		gotConsensusNodes, err := chain.getConsensusNodes(&c.prevBlockHash)
		if err != nil {
			t.Fatal(err)
		}

		wantConsensusNodes := make(map[string]*state.ConsensusNode)
		for _, node := range c.wantConsensusNodes {
			wantConsensusNodes[node.XPub.String()] = node
		}

		if !testutil.DeepEqual(gotConsensusNodes, wantConsensusNodes) {
			t.Errorf("#%d (%s) got consensus nodes:%v, want consensus nodes:%v", i, c.desc, gotConsensusNodes, wantConsensusNodes)
		}
	}
}

func mustDecodeXPub(xpub string) chainkd.XPub {
	bytes := testutil.MustDecodeHexString(xpub)
	var result [64]byte
	copy(result[:], bytes)
	return result
}

type dummyStore struct {
	blockHeaders     map[string]*types.BlockHeader
	blocks           map[string]*types.Block
	consensusResults map[uint64]*state.ConsensusResult
}

func newDummyStore() *dummyStore {
	return &dummyStore{
		blockHeaders:     make(map[string]*types.BlockHeader),
		consensusResults: make(map[uint64]*state.ConsensusResult),
		blocks:           make(map[string]*types.Block),
	}
}

func (s *dummyStore) BlockExist(*bc.Hash) bool {
	return false
}

func (s *dummyStore) GetBlock(hash *bc.Hash) (*types.Block, error) {
	return s.blocks[hash.String()], nil
}

func (s *dummyStore) GetBlockHeader(hash *bc.Hash) (*types.BlockHeader, error) {
	return s.blockHeaders[hash.String()], nil
}

func (s *dummyStore) GetStoreStatus() *BlockStoreState {
	return nil
}

func (s *dummyStore) GetTransactionStatus(*bc.Hash) (*bc.TransactionStatus, error) {
	return nil, nil
}

func (s *dummyStore) GetTransactionsUtxo(*state.UtxoViewpoint, []*bc.Tx) error {
	return nil
}

func (s *dummyStore) GetUtxo(*bc.Hash) (*storage.UtxoEntry, error) {
	return nil, nil
}

func (s *dummyStore) GetConsensusResult(seq uint64) (*state.ConsensusResult, error) {
	return s.consensusResults[seq], nil
}

func (s *dummyStore) SetConsensusResult(consensusResult *state.ConsensusResult) {
	s.consensusResults[consensusResult.Seq] = consensusResult
}

func (s *dummyStore) GetMainChainHash(uint64) (*bc.Hash, error) {
	return nil, nil
}

func (s *dummyStore) GetBlockHashesByHeight(uint64) ([]*bc.Hash, error) {
	return nil, nil
}

func (s *dummyStore) DeleteConsensusResult(seq uint64) error {
	return nil
}

func (s *dummyStore) SaveBlock(block *types.Block, _ *bc.TransactionStatus) error {
	hash := block.Hash()
	s.blocks[hash.String()] = block
	return nil
}

func (s *dummyStore) DeleteBlock(block *types.Block) error {
	return nil
}

func (s *dummyStore) SaveBlockHeader(header *types.BlockHeader) error {
	hash := header.Hash()
	s.blockHeaders[hash.String()] = header
	return nil
}

func (s *dummyStore) SaveChainStatus(*types.BlockHeader, *types.BlockHeader, []*types.BlockHeader, *state.UtxoViewpoint, []*state.ConsensusResult) error {
	return nil
}
