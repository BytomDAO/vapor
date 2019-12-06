package state

import (
	"encoding/hex"
	"math"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/crypto/ed25519/chainkd"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/math/checked"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/testutil"
)

func TestApplyTransaction(t *testing.T) {
	testXpub, _ := hex.DecodeString("a8018a1ba4d85fc7118bbd065612da78b2c503e61a1a093d9c659567c5d3a591b3752569fbcafa951b2304b8f576f3f220e03b957ca819840e7c29e4b7fb2c4d")

	cases := []struct {
		desc                string
		tx                  *types.Tx
		prevConsensusResult *ConsensusResult
		postConsensusResult *ConsensusResult
		wantErr             error
	}{
		{
			desc: "test num Of vote overflow",
			tx: &types.Tx{
				TxData: types.TxData{
					Inputs:  []*types.TxInput{types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, 600000000, 0, nil)},
					Outputs: []*types.TxOutput{types.NewVoteOutput(*consensus.BTMAssetID, math.MaxUint64-1000, []byte{0x51}, testXpub)},
				},
			},
			prevConsensusResult: &ConsensusResult{
				NumOfVote: map[string]uint64{
					hex.EncodeToString(testXpub): 1000000,
				},
			},
			postConsensusResult: &ConsensusResult{
				NumOfVote: map[string]uint64{},
			},
			wantErr: checked.ErrOverflow,
		},
		{
			desc: "test num Of veto overflow",
			tx: &types.Tx{
				TxData: types.TxData{
					Inputs:  []*types.TxInput{types.NewVetoInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, 100000000, 0, []byte{0x51}, testXpub)},
					Outputs: []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 100000000, []byte{0x51})},
				},
			},
			prevConsensusResult: &ConsensusResult{
				NumOfVote: map[string]uint64{
					hex.EncodeToString(testXpub): 1000000,
				},
			},
			postConsensusResult: &ConsensusResult{
				NumOfVote: map[string]uint64{},
			},
			wantErr: checked.ErrOverflow,
		},
		{
			desc: "test del pubkey from NumOfVote",
			tx: &types.Tx{
				TxData: types.TxData{
					Inputs:  []*types.TxInput{types.NewVetoInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, 1000000, 0, []byte{0x51}, testXpub)},
					Outputs: []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 100000000, []byte{0x51})},
				},
			},
			prevConsensusResult: &ConsensusResult{
				NumOfVote: map[string]uint64{
					hex.EncodeToString(testXpub): 1000000,
				},
			},
			postConsensusResult: &ConsensusResult{
				NumOfVote: map[string]uint64{},
			},
			wantErr: nil,
		},
	}

	for i, c := range cases {
		if err := c.prevConsensusResult.ApplyTransaction(c.tx); err != nil {
			if err != c.wantErr {
				t.Errorf("test case #%d want err = %v, got err = %v", i, c.wantErr, err)
			}
			continue
		}

		if !testutil.DeepEqual(c.prevConsensusResult, c.postConsensusResult) {
			t.Errorf("test case #%d, want %v, got %v", i, c.postConsensusResult, c.prevConsensusResult)
		}
	}
}

func TestAttachCoinbaseReward(t *testing.T) {
	cases := []struct {
		desc                string
		block               *types.Block
		prevConsensusResult *ConsensusResult
		postConsensusResult *ConsensusResult
		wantErr             error
	}{
		{
			desc: "normal test with block contain coinbase tx and other tx",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height: 1,
				},
				Transactions: []*types.Tx{
					{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						},
					},
					{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, 300000000, 0, nil)},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 250000000, []byte{0x51})},
						},
					},
				},
			},
			prevConsensusResult: &ConsensusResult{
				CoinbaseReward: map[string]uint64{
					hex.EncodeToString([]byte{0x51}): 50000000,
					hex.EncodeToString([]byte{0x52}): 80000000,
				},
			},
			postConsensusResult: &ConsensusResult{
				CoinbaseReward: map[string]uint64{
					hex.EncodeToString([]byte{0x51}): consensus.BlockSubsidy(1) + 50000000,
				},
			},
			wantErr: nil,
		},
		{
			desc: "test coinbase reward overflow",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height: 100,
				},
				Transactions: []*types.Tx{
					{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						},
					},
					{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, math.MaxUint64-80000000, 0, nil)},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 0, []byte{0x52})},
						},
					},
				},
			},
			prevConsensusResult: &ConsensusResult{
				CoinbaseReward: map[string]uint64{
					hex.EncodeToString([]byte{0x51}): 80000000,
					hex.EncodeToString([]byte{0x52}): 50000000,
				},
			},
			postConsensusResult: &ConsensusResult{
				CoinbaseReward: map[string]uint64{
					hex.EncodeToString([]byte{0x51}): consensus.BlockSubsidy(1) + 50000000,
				},
			},
			wantErr: checked.ErrOverflow,
		},
	}

	for i, c := range cases {
		if err := c.prevConsensusResult.AttachCoinbaseReward(c.block); err != nil {
			if err != c.wantErr {
				t.Errorf("test case #%d want err = %v, got err = %v", i, c.wantErr, err)
			}
			continue
		}

		if !testutil.DeepEqual(c.prevConsensusResult, c.postConsensusResult) {
			t.Errorf("test case #%d, want %v, got %v", i, c.postConsensusResult, c.prevConsensusResult)
		}
	}
}

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
		{
			desc: "test number of vote overflow",
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
				BlockHash: testutil.MustDecodeHash("4ebd9e7c00d3e0370931689c6eb9e2131c6700fe66e6b9718028dd75d7a4e329"),
				CoinbaseReward: map[string]uint64{
					"51": 100000000,
				},
				NumOfVote: map[string]uint64{},
			},
			wantErr: checked.ErrOverflow,
		},
		{
			desc: "test number of veto overflow",
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
							Inputs:  []*types.TxInput{types.NewVetoInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, math.MaxUint64, 0, []byte{0x51}, testXpub)},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, math.MaxUint64, []byte{0x51})},
						},
					},
				},
			},
			consensusResult: &ConsensusResult{
				BlockHash: testutil.MustDecodeHash("4ebd9e7c00d3e0370931689c6eb9e2131c6700fe66e6b9718028dd75d7a4e329"),
				CoinbaseReward: map[string]uint64{
					"51": 100000000,
				},
				NumOfVote: map[string]uint64{
					"a8018a1ba4d85fc7118bbd065612da78b2c503e61a1a093d9c659567c5d3a591b3752569fbcafa951b2304b8f576f3f220e03b957ca819840e7c29e4b7fb2c4d": 100,
				},
			},
			wantErr: checked.ErrOverflow,
		},
		{
			desc: "test detch coinbase overflow",
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
							Inputs:  []*types.TxInput{types.NewVetoInput(nil, bc.NewHash([32]byte{0xff}), *consensus.BTMAssetID, math.MaxUint64, 0, []byte{0x51}, testXpub)},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, math.MaxUint64, []byte{0x51})},
						},
					},
				},
			},
			consensusResult: &ConsensusResult{
				BlockHash:      testutil.MustDecodeHash("4ebd9e7c00d3e0370931689c6eb9e2131c6700fe66e6b9718028dd75d7a4e329"),
				CoinbaseReward: map[string]uint64{},
				NumOfVote:      map[string]uint64{},
			},
			wantErr: checked.ErrOverflow,
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

func TestConsensusNodes(t *testing.T) {
	var xpub1, xpub2, xpub3, xpub4, xpub5, xpub6, xpub7 chainkd.XPub
	strPub1 := "0f8669abbd3cc0a167156188e428f940088d5b2f36bb3449df71d2bdc5e077814ea3f68628eef279ed435f51ee26cff00f8bd28fabfd500bedb2a9e369f5c825"
	strPub2 := "e7f458ee8d2ba19b0fdc7410d1fd57e9c2e1a79377c661d66c55effe49d7ffc920e40510442d4a10b7bea06c09fb0b41f52601135adaaa7136204db36106c093"
	strPub3 := "1bec3a35da038ec7a76c40986e80b5af2dcef60341970e3fc58b4db0797bd4ca9b2cbf3d7ab820832e22a80b5b86ae1427f7f706a7780089958b2862e7bc0842"
	strPub4 := "b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9"
	strPub5 := "b928e46bb01e834fdf167185e31b15de7cc257af8bbdf17f9c7fefd5bb97b306d048b6bc0da2097152c1c2ff38333c756a543adbba7030a447dcc776b8ac64ef"
	strPub6 := "36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67"
	strPub7 := "123"

	xpub1.UnmarshalText([]byte(strPub1))
	xpub2.UnmarshalText([]byte(strPub2))
	xpub3.UnmarshalText([]byte(strPub3))
	xpub4.UnmarshalText([]byte(strPub4))
	xpub5.UnmarshalText([]byte(strPub5))
	xpub6.UnmarshalText([]byte(strPub6))
	xpub7.UnmarshalText([]byte(strPub7))

	cases := []struct {
		consensusResult *ConsensusResult
		consensusNode   map[string]*ConsensusNode
		wantErr         error
	}{
		{
			consensusResult: &ConsensusResult{
				NumOfVote: map[string]uint64{
					strPub1: 838063475500000,  //1
					strPub2: 474794800000000,  //3
					strPub3: 833812985000000,  //2
					strPub4: 285918061999999,  //4
					strPub5: 1228455289930297, //0
					strPub6: 274387690000000,  //5
					strPub7: 1028455289930297,
				},
			},
			consensusNode: map[string]*ConsensusNode{
				strPub1: &ConsensusNode{XPub: xpub1, VoteNum: 838063475500000, Order: 1},
				strPub2: &ConsensusNode{XPub: xpub2, VoteNum: 474794800000000, Order: 3},
				strPub3: &ConsensusNode{XPub: xpub3, VoteNum: 833812985000000, Order: 2},
				strPub4: &ConsensusNode{XPub: xpub4, VoteNum: 285918061999999, Order: 4},
				strPub5: &ConsensusNode{XPub: xpub5, VoteNum: 1228455289930297, Order: 0},
				strPub6: &ConsensusNode{XPub: xpub6, VoteNum: 274387690000000, Order: 5},
			},
			wantErr: chainkd.ErrBadKeyStr,
		},
	}

	for i, c := range cases {
		consensusNode, err := c.consensusResult.ConsensusNodes()
		if err != nil {
			if err != c.wantErr {
				t.Errorf("test case #%d want err = %v, got err = %v", i, c.wantErr, err)
			}
			continue
		}

		if !testutil.DeepEqual(consensusNode, c.consensusNode) {
			t.Errorf("test case #%d, want %v, got %v", i, c.consensusNode, consensusNode)
		}
	}
}

func TestFork(t *testing.T) {
	consensusResult := &ConsensusResult{
		Seq: 100,
		NumOfVote: map[string]uint64{
			"a": 100,
			"b": 200,
		},
		CoinbaseReward: map[string]uint64{
			"c": 300,
			"d": 400,
		},
		BlockHash:   bc.NewHash([32]byte{0x1, 0x2}),
		BlockHeight: 1024,
	}
	copy := consensusResult.Fork()

	if !reflect.DeepEqual(consensusResult, copy) {
		t.Fatalf("failed on test consensusResult got %s want %s", spew.Sdump(copy), spew.Sdump(consensusResult))
	}
}
