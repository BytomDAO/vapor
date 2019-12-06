package state

import (
	"encoding/hex"
	"testing"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/math/checked"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/testutil"
)

func TestCalCoinbaseReward(t *testing.T) {
	cases := []struct {
		desc       string
		block      *types.Block
		wantReward *CoinbaseReward
		wantErr    error
	}{
		{
			desc: "normal test with block contain coinbase tx and other tx",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height: 1,
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						},
					},
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, 300000000, 0, nil)},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 250000000, []byte{0x51})},
						},
					},
				},
			},
			wantReward: &CoinbaseReward{
				Amount:         consensus.BlockSubsidy(1) + 50000000,
				ControlProgram: []byte{0x51},
			},
		},
		{
			desc: "normal test with block only contain coinbase tx",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height: 1200,
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 1000000000, []byte{0x51})},
						},
					},
				},
			},
			wantReward: &CoinbaseReward{
				Amount:         consensus.BlockSubsidy(1),
				ControlProgram: []byte{0x51},
			},
		},
		{
			desc: "abnormal test with block not contain coinbase tx",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height: 1,
				},
				Transactions: []*types.Tx{},
			},
			wantErr: errors.New("not found coinbase receiver"),
		},
	}

	for i, c := range cases {
		coinbaseReward, err := CalCoinbaseReward(c.block)
		if err != nil {
			if err.Error() != c.wantErr.Error() {
				t.Errorf("test case #%d want err = %v, got err = %v", i, c.wantErr, err)
			}
			continue
		}

		if !testutil.DeepEqual(coinbaseReward, c.wantReward) {
			t.Errorf("test case #%d, want %v, got %v", i, c.wantReward, coinbaseReward)
		}
	}
}

func TestConsensusApplyBlock(t *testing.T) {
	testXpub, _ := hex.DecodeString("a8018a1ba4d85fc7118bbd065612da78b2c503e61a1a093d9c659567c5d3a591b3752569fbcafa951b2304b8f576f3f220e03b957ca819840e7c29e4b7fb2c4d")
	cases := []struct {
		desc            string
		block           *types.Block
		consensusResult *ConsensusResult
		wantResult      *ConsensusResult
		wantErr         error
	}{
		{
			desc: "normal test with block height is equal to 1",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            1,
					PreviousBlockHash: bc.Hash{V0: 1},
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						},
					},
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, 300000000, 0, nil)},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 250000000, []byte{0x51})},
						},
					},
				},
			},
			consensusResult: &ConsensusResult{
				Seq:            0,
				NumOfVote:      map[string]uint64{},
				CoinbaseReward: map[string]uint64{},
				BlockHash:      bc.Hash{V0: 1},
				BlockHeight:    0,
			},
			wantResult: &ConsensusResult{
				Seq:       1,
				NumOfVote: map[string]uint64{},
				CoinbaseReward: map[string]uint64{
					"51": consensus.BlockSubsidy(1) + 50000000,
				},
				BlockHash:   testutil.MustDecodeHash("50da6990965a16e97b739accca7eb8a0fadd47c8a742f77d18fa51ab60dd8724"),
				BlockHeight: 1,
			},
		},
		{
			desc: "normal test with block height is equal to RoundVoteBlockNums - 1",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            consensus.MainNetParams.RoundVoteBlockNums - 1,
					PreviousBlockHash: bc.Hash{V0: 1},
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						},
					},
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, 600000000, 0, nil)},
							Outputs: []*types.TxOutput{types.NewVoteOutput(*consensus.BTMAssetID, 600000000, []byte{0x51}, testXpub)},
						},
					},
				},
			},
			consensusResult: &ConsensusResult{
				NumOfVote:      map[string]uint64{},
				CoinbaseReward: map[string]uint64{},
				BlockHash:      bc.Hash{V0: 1},
				BlockHeight:    consensus.MainNetParams.RoundVoteBlockNums - 2,
			},
			wantResult: &ConsensusResult{
				Seq: 1,
				NumOfVote: map[string]uint64{
					"a8018a1ba4d85fc7118bbd065612da78b2c503e61a1a093d9c659567c5d3a591b3752569fbcafa951b2304b8f576f3f220e03b957ca819840e7c29e4b7fb2c4d": 600000000,
				},
				CoinbaseReward: map[string]uint64{
					"51": consensus.BlockSubsidy(consensus.MainNetParams.RoundVoteBlockNums - 1),
				},
				BlockHash:   testutil.MustDecodeHash("4ebd9e7c00d3e0370931689c6eb9e2131c6700fe66e6b9718028dd75d7a4e329"),
				BlockHeight: consensus.MainNetParams.RoundVoteBlockNums - 1,
			},
		},
		{
			desc: "normal test with block height is equal to RoundVoteBlockNums",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            consensus.MainNetParams.RoundVoteBlockNums,
					PreviousBlockHash: bc.Hash{V0: 1},
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs: []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{
								types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51}),
							},
						},
					},
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, 600000000, 0, nil)},
							Outputs: []*types.TxOutput{types.NewVoteOutput(*consensus.BTMAssetID, 600000000, []byte{0x51}, testXpub)},
						},
					},
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewVetoInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, 100000000, 0, []byte{0x51}, testXpub)},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 100000000, []byte{0x51})},
						},
					},
				},
			},
			consensusResult: &ConsensusResult{
				NumOfVote: map[string]uint64{},
				CoinbaseReward: map[string]uint64{
					"51": 10000000,
					"52": 20000000,
					"53": 30000000,
				},
				BlockHash:   bc.Hash{V0: 1},
				BlockHeight: consensus.MainNetParams.RoundVoteBlockNums - 1,
			},
			wantResult: &ConsensusResult{
				Seq: 1,
				NumOfVote: map[string]uint64{
					"a8018a1ba4d85fc7118bbd065612da78b2c503e61a1a093d9c659567c5d3a591b3752569fbcafa951b2304b8f576f3f220e03b957ca819840e7c29e4b7fb2c4d": 500000000,
				},
				CoinbaseReward: map[string]uint64{
					"51": consensus.BlockSubsidy(consensus.MainNetParams.RoundVoteBlockNums) + 10000000,
					"52": 20000000,
					"53": 30000000,
				},
				BlockHash:   testutil.MustDecodeHash("1b449ba1f9b0ae41e31238b32943b95e9ab292d0b4a93d690ef9bc689c31d362"),
				BlockHeight: consensus.MainNetParams.RoundVoteBlockNums,
			},
		},
		{
			desc: "normal test with block height is equal to RoundVoteBlockNums + 1",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            consensus.MainNetParams.RoundVoteBlockNums + 1,
					PreviousBlockHash: bc.Hash{V0: 1},
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs: []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{
								types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51}),
								types.NewIntraChainOutput(bc.AssetID{}, consensus.BlockSubsidy(consensus.MainNetParams.RoundVoteBlockNums)+10000000, []byte{0x51}),
								types.NewIntraChainOutput(bc.AssetID{}, 20000000, []byte{0x53}),
								types.NewIntraChainOutput(bc.AssetID{}, 30000000, []byte{0x52}),
							},
						},
					},
				},
			},
			consensusResult: &ConsensusResult{
				NumOfVote: map[string]uint64{},
				CoinbaseReward: map[string]uint64{
					"51": consensus.BlockSubsidy(consensus.MainNetParams.RoundVoteBlockNums) + 10000000,
					"52": 20000000,
					"53": 30000000,
				},
				BlockHash:   bc.Hash{V0: 1},
				BlockHeight: consensus.MainNetParams.RoundVoteBlockNums,
			},
			wantResult: &ConsensusResult{
				Seq:       2,
				NumOfVote: map[string]uint64{},
				CoinbaseReward: map[string]uint64{
					"51": consensus.BlockSubsidy(consensus.MainNetParams.RoundVoteBlockNums + 1),
				},
				BlockHash:   testutil.MustDecodeHash("52681d209ab811359f92daaf46a771ecd0f28505ae5e0ac2f0feb80f76fdda59"),
				BlockHeight: consensus.MainNetParams.RoundVoteBlockNums + 1,
			},
		},
		{
			desc: "normal test with block height is equal to RoundVoteBlockNums + 2",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            consensus.MainNetParams.RoundVoteBlockNums + 2,
					PreviousBlockHash: bc.Hash{V0: 1},
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs: []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{
								types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51}),
							},
						},
					},
				},
			},
			consensusResult: &ConsensusResult{
				NumOfVote: map[string]uint64{},
				CoinbaseReward: map[string]uint64{
					"51": consensus.BlockSubsidy(consensus.MainNetParams.RoundVoteBlockNums + 1),
				},
				BlockHash:   bc.Hash{V0: 1},
				BlockHeight: consensus.MainNetParams.RoundVoteBlockNums + 1,
			},
			wantResult: &ConsensusResult{
				Seq:       2,
				NumOfVote: map[string]uint64{},
				CoinbaseReward: map[string]uint64{
					"51": consensus.BlockSubsidy(consensus.MainNetParams.RoundVoteBlockNums+1) + consensus.BlockSubsidy(consensus.MainNetParams.RoundVoteBlockNums+2),
				},
				BlockHash:   testutil.MustDecodeHash("3de69f8af48b77e81232c71d30b25dd4ac482be45402a0fd417a4a040c135b76"),
				BlockHeight: consensus.MainNetParams.RoundVoteBlockNums + 2,
			},
		},
		{
			desc: "abnormal test with block parent hash is not equals last block hash of vote result",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            1,
					PreviousBlockHash: bc.Hash{V0: 0},
				},
				Transactions: []*types.Tx{},
			},
			consensusResult: &ConsensusResult{
				NumOfVote:      map[string]uint64{},
				CoinbaseReward: map[string]uint64{},
				BlockHash:      bc.Hash{V0: 1},
				BlockHeight:    2,
			},
			wantErr: errors.New("block parent hash is not equals last block hash of vote result"),
		},
		{
			desc: "abnormal test with arithmetic overflow for calculate transaction fee",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            1,
					PreviousBlockHash: bc.Hash{V0: 1},
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						},
					},
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, 100000000, 0, nil)},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 200000000, []byte{0x51})},
						},
					},
				},
			},
			consensusResult: &ConsensusResult{
				NumOfVote:      map[string]uint64{},
				CoinbaseReward: map[string]uint64{},
				BlockHash:      bc.Hash{V0: 1},
				BlockHeight:    2,
			},
			wantErr: errors.Wrap(checked.ErrOverflow, "calculate transaction fee"),
		},
		{
			desc: "abnormal test with not found coinbase receiver",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            1,
					PreviousBlockHash: bc.Hash{V0: 1},
				},
				Transactions: []*types.Tx{},
			},
			consensusResult: &ConsensusResult{
				NumOfVote:      map[string]uint64{},
				CoinbaseReward: map[string]uint64{},
				BlockHash:      bc.Hash{V0: 1},
				BlockHeight:    2,
			},
			wantErr: errors.New("not found coinbase receiver"),
		},
	}

	for i, c := range cases {
		if err := c.consensusResult.ApplyBlock(c.block); err != nil {
			if err.Error() != c.wantErr.Error() {
				t.Errorf("test case #%d want err = %v, got err = %v", i, c.wantErr, err)
			}
			continue
		}

		if !testutil.DeepEqual(c.consensusResult, c.wantResult) {
			t.Errorf("test case #%d, want %v, got %v", i, c.wantResult, c.consensusResult)
		}
	}
}

func TestConsensusDetachBlock(t *testing.T) {
	testXpub, _ := hex.DecodeString("a8018a1ba4d85fc7118bbd065612da78b2c503e61a1a093d9c659567c5d3a591b3752569fbcafa951b2304b8f576f3f220e03b957ca819840e7c29e4b7fb2c4d")
	cases := []struct {
		desc            string
		block           *types.Block
		consensusResult *ConsensusResult
		wantResult      *ConsensusResult
		wantErr         error
	}{
		{
			desc: "normal test with block height is equal to 1",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            1,
					PreviousBlockHash: bc.Hash{V0: 1},
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						},
					},
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, 300000000, 0, nil)},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 250000000, []byte{0x51})},
						},
					},
				},
			},
			consensusResult: &ConsensusResult{
				Seq:       1,
				NumOfVote: map[string]uint64{},
				CoinbaseReward: map[string]uint64{
					"51": consensus.BlockSubsidy(1) + 50000000,
				},
				BlockHash:   testutil.MustDecodeHash("50da6990965a16e97b739accca7eb8a0fadd47c8a742f77d18fa51ab60dd8724"),
				BlockHeight: 1,
			},
			wantResult: &ConsensusResult{
				Seq:            0,
				NumOfVote:      map[string]uint64{},
				CoinbaseReward: map[string]uint64{},
				BlockHash:      bc.Hash{V0: 1},
				BlockHeight:    0,
			},
		},
		{
			desc: "normal test with block height is equal to RoundVoteBlockNums - 1",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            consensus.MainNetParams.RoundVoteBlockNums - 1,
					PreviousBlockHash: bc.Hash{V0: 1},
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						},
					},
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, 600000000, 0, nil)},
							Outputs: []*types.TxOutput{types.NewVoteOutput(*consensus.BTMAssetID, 600000000, []byte{0x51}, testXpub)},
						},
					},
				},
			},
			consensusResult: &ConsensusResult{
				NumOfVote: map[string]uint64{
					"a8018a1ba4d85fc7118bbd065612da78b2c503e61a1a093d9c659567c5d3a591b3752569fbcafa951b2304b8f576f3f220e03b957ca819840e7c29e4b7fb2c4d": 600000000,
				},
				CoinbaseReward: map[string]uint64{
					"51": consensus.BlockSubsidy(consensus.MainNetParams.RoundVoteBlockNums - 1),
				},
				BlockHash:   testutil.MustDecodeHash("4ebd9e7c00d3e0370931689c6eb9e2131c6700fe66e6b9718028dd75d7a4e329"),
				BlockHeight: consensus.MainNetParams.RoundVoteBlockNums - 1,
			},
			wantResult: &ConsensusResult{
				Seq:            1,
				NumOfVote:      map[string]uint64{},
				CoinbaseReward: map[string]uint64{},
				BlockHash:      bc.Hash{V0: 1},
				BlockHeight:    consensus.MainNetParams.RoundVoteBlockNums - 2,
			},
		},
		{
			desc: "normal test with block height is equal to RoundVoteBlockNums",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            consensus.MainNetParams.RoundVoteBlockNums,
					PreviousBlockHash: bc.Hash{V0: 1},
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs: []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{
								types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51}),
							},
						},
					},
				},
			},
			consensusResult: &ConsensusResult{
				NumOfVote: map[string]uint64{
					"a8018a1ba4d85fc7118bbd065612da78b2c503e61a1a093d9c659567c5d3a591b3752569fbcafa951b2304b8f576f3f220e03b957ca819840e7c29e4b7fb2c4d": 500000000,
				},
				CoinbaseReward: map[string]uint64{
					"51": consensus.BlockSubsidy(consensus.MainNetParams.RoundVoteBlockNums) + 100000000,
				},
				BlockHash:   testutil.MustDecodeHash("1b449ba1f9b0ae41e31238b32943b95e9ab292d0b4a93d690ef9bc689c31d362"),
				BlockHeight: consensus.MainNetParams.RoundVoteBlockNums,
			},
			wantResult: &ConsensusResult{
				Seq: 1,
				NumOfVote: map[string]uint64{
					"a8018a1ba4d85fc7118bbd065612da78b2c503e61a1a093d9c659567c5d3a591b3752569fbcafa951b2304b8f576f3f220e03b957ca819840e7c29e4b7fb2c4d": 500000000,
				},
				CoinbaseReward: map[string]uint64{
					"51": 100000000,
				},
				BlockHash:   bc.Hash{V0: 1},
				BlockHeight: consensus.MainNetParams.RoundVoteBlockNums - 1,
			},
		},
		{
			desc: "normal test with block height is equal to RoundVoteBlockNums + 1",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            consensus.MainNetParams.RoundVoteBlockNums + 1,
					PreviousBlockHash: bc.Hash{V0: 1},
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs: []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{
								types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51}),
								types.NewIntraChainOutput(bc.AssetID{}, consensus.BlockSubsidy(consensus.MainNetParams.RoundVoteBlockNums)+10000000, []byte{0x51}),
								types.NewIntraChainOutput(bc.AssetID{}, 20000000, []byte{0x52}),
							},
						},
					},
				},
			},
			consensusResult: &ConsensusResult{
				NumOfVote: map[string]uint64{},
				CoinbaseReward: map[string]uint64{
					"51": consensus.BlockSubsidy(consensus.MainNetParams.RoundVoteBlockNums + 1),
				},
				BlockHash:   testutil.MustDecodeHash("52681d209ab811359f92daaf46a771ecd0f28505ae5e0ac2f0feb80f76fdda59"),
				BlockHeight: consensus.MainNetParams.RoundVoteBlockNums,
			},
			wantResult: &ConsensusResult{
				Seq:       1,
				NumOfVote: map[string]uint64{},
				CoinbaseReward: map[string]uint64{
					"51": consensus.BlockSubsidy(consensus.MainNetParams.RoundVoteBlockNums) + 10000000,
					"52": 20000000,
				},
				BlockHash:   bc.Hash{V0: 1},
				BlockHeight: consensus.MainNetParams.RoundVoteBlockNums,
			},
		},
		{
			desc: "normal test with block height is equal to RoundVoteBlockNums + 2",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            consensus.MainNetParams.RoundVoteBlockNums + 2,
					PreviousBlockHash: bc.Hash{V0: 1},
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs: []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{
								types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51}),
							},
						},
					},
				},
			},
			consensusResult: &ConsensusResult{
				NumOfVote: map[string]uint64{},
				CoinbaseReward: map[string]uint64{
					"51": consensus.BlockSubsidy(consensus.MainNetParams.RoundVoteBlockNums+1) + 1000000,
				},
				BlockHash:   testutil.MustDecodeHash("3de69f8af48b77e81232c71d30b25dd4ac482be45402a0fd417a4a040c135b76"),
				BlockHeight: consensus.MainNetParams.RoundVoteBlockNums + 1,
			},
			wantResult: &ConsensusResult{
				Seq:       2,
				NumOfVote: map[string]uint64{},
				CoinbaseReward: map[string]uint64{
					"51": 1000000,
				},
				BlockHash:   bc.Hash{V0: 1},
				BlockHeight: consensus.MainNetParams.RoundVoteBlockNums + 1,
			},
		},
		{
			desc: "abnormal test with block hash is not equals last block hash of vote result",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            2,
					PreviousBlockHash: bc.Hash{V0: 0},
				},
				Transactions: []*types.Tx{},
			},
			consensusResult: &ConsensusResult{
				NumOfVote:      map[string]uint64{},
				CoinbaseReward: map[string]uint64{},
				BlockHash:      bc.Hash{V0: 1},
				BlockHeight:    1,
			},
			wantErr: errors.New("block hash is not equals last block hash of vote result"),
		},
		{
			desc: "abnormal test with arithmetic overflow for calculate transaction fee",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            2,
					PreviousBlockHash: bc.Hash{V0: 1},
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						},
					},
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, 100000000, 0, nil)},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 200000000, []byte{0x51})},
						},
					},
				},
			},
			consensusResult: &ConsensusResult{
				NumOfVote:      map[string]uint64{},
				CoinbaseReward: map[string]uint64{},
				BlockHash:      testutil.MustDecodeHash("02b7fb48defc4f4a3e1ef8403f7c0be78c4414ee66aa81fd702caa1e41a906df"),
				BlockHeight:    1,
			},
			wantErr: errors.Wrap(checked.ErrOverflow, "calculate transaction fee"),
		},
		{
			desc: "abnormal test with not found coinbase receiver",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height:            1,
					PreviousBlockHash: bc.Hash{V0: 1},
				},
				Transactions: []*types.Tx{},
			},
			consensusResult: &ConsensusResult{
				NumOfVote:      map[string]uint64{},
				CoinbaseReward: map[string]uint64{},
				BlockHash:      testutil.MustDecodeHash("50da6990965a16e97b739accca7eb8a0fadd47c8a742f77d18fa51ab60dd8724"),
				BlockHeight:    1,
			},
			wantErr: errors.New("not found coinbase receiver"),
		},
	}

	for i, c := range cases {
		if err := c.consensusResult.DetachBlock(c.block); err != nil {
			if err.Error() != c.wantErr.Error() {
				t.Errorf("test case #%d want err = %v, got err = %v", i, c.wantErr, err)
			}
			continue
		}

		if !testutil.DeepEqual(c.consensusResult, c.wantResult) {
			t.Errorf("test case #%d, want %v, got %v", i, c.wantResult, c.consensusResult)
		}
	}
}

func TestGetCoinbaseRewards(t *testing.T) {
	cases := []struct {
		desc            string
		blockHeight     uint64
		consensusResult *ConsensusResult
		wantRewards     []CoinbaseReward
	}{
		{
			desc:        "the block height is RoundVoteBlockNums - 1",
			blockHeight: consensus.ActiveNetParams.RoundVoteBlockNums - 1,
			consensusResult: &ConsensusResult{
				CoinbaseReward: map[string]uint64{
					"51": 10,
				},
			},
			wantRewards: []CoinbaseReward{},
		},
		{
			desc:        "the block height is RoundVoteBlockNums",
			blockHeight: consensus.ActiveNetParams.RoundVoteBlockNums,
			consensusResult: &ConsensusResult{
				CoinbaseReward: map[string]uint64{
					"51": 10,
					"52": 20,
				},
			},
			wantRewards: []CoinbaseReward{
				CoinbaseReward{
					Amount:         20,
					ControlProgram: []byte{0x52},
				},
				CoinbaseReward{
					Amount:         10,
					ControlProgram: []byte{0x51},
				},
			},
		},
		{
			desc:        "the block height is RoundVoteBlockNums + 1",
			blockHeight: consensus.ActiveNetParams.RoundVoteBlockNums + 1,
			consensusResult: &ConsensusResult{
				CoinbaseReward: map[string]uint64{
					"51": 10,
				},
			},
			wantRewards: []CoinbaseReward{},
		},
		{
			desc:        "the block height is RoundVoteBlockNums * 2",
			blockHeight: consensus.ActiveNetParams.RoundVoteBlockNums * 2,
			consensusResult: &ConsensusResult{
				CoinbaseReward: map[string]uint64{
					"50": 20,
					"51": 10,
					"52": 20,
					"53": 30,
				},
			},
			wantRewards: []CoinbaseReward{
				CoinbaseReward{
					Amount:         30,
					ControlProgram: []byte{0x53},
				},
				CoinbaseReward{
					Amount:         20,
					ControlProgram: []byte{0x52},
				},
				CoinbaseReward{
					Amount:         20,
					ControlProgram: []byte{0x50},
				},
				CoinbaseReward{
					Amount:         10,
					ControlProgram: []byte{0x51},
				},
			},
		},
		{
			desc:        "the block height is 2*RoundVoteBlockNums + 1",
			blockHeight: 2*consensus.ActiveNetParams.RoundVoteBlockNums + 1,
			consensusResult: &ConsensusResult{
				CoinbaseReward: map[string]uint64{},
			},
			wantRewards: []CoinbaseReward{},
		},
	}

	for i, c := range cases {
		rewards, err := c.consensusResult.GetCoinbaseRewards(c.blockHeight)
		if err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(rewards, c.wantRewards) {
			t.Errorf("test case #%d, want %v, got %v", i, c.wantRewards, rewards)
		}
	}
}
