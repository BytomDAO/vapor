package proposal

import (
	"testing"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
	"github.com/bytom/vapor/testutil"
)

func TestCalCoinbaseTxReward(t *testing.T) {
	consensus.ActiveNetParams.ProducerSubsidys = []consensus.ProducerSubsidy{
		{BeginBlock: 0, EndBlock: 0, Subsidy: 24},
		{BeginBlock: 1, EndBlock: 840000, Subsidy: 24},
		{BeginBlock: 840001, EndBlock: 1680000, Subsidy: 12},
		{BeginBlock: 1680001, EndBlock: 3360000, Subsidy: 6},
	}
	reductionInterval := uint64(840000)

	cases := []struct {
		desc       string
		block      *types.Block
		wantReward state.CoinbaseReward
	}{
		{
			desc: "the block height is reductionInterval - 1",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height: reductionInterval - 1,
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						},
					},
				},
			},
			wantReward: state.CoinbaseReward{
				Amount:         24,
				ControlProgram: []byte{0x51},
			},
		},
		{
			desc: "the block height is reductionInterval",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height: reductionInterval,
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						},
					},
				},
			},
			wantReward: state.CoinbaseReward{
				Amount:         24,
				ControlProgram: []byte{0x51},
			},
		},
		{
			desc: "the block height is reductionInterval + 1",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height: reductionInterval + 1,
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						},
					},
				},
			},
			wantReward: state.CoinbaseReward{
				Amount:         12,
				ControlProgram: []byte{0x51},
			},
		},
		{
			desc: "the block height is reductionInterval * 2",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height: reductionInterval * 2,
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						},
					},
				},
			},
			wantReward: state.CoinbaseReward{
				Amount:         12,
				ControlProgram: []byte{0x51},
			},
		},
		{
			desc: "the block height is 2*reductionInterval + 1",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height: 2*reductionInterval + 1,
				},
				Transactions: []*types.Tx{
					&types.Tx{
						TxData: types.TxData{
							Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
							Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
						},
					},
				},
			},
			wantReward: state.CoinbaseReward{
				Amount:         6,
				ControlProgram: []byte{0x51},
			},
		},
	}

	for i, c := range cases {
		gotReward, err := state.CalCoinbaseReward(c.block)
		if err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(*gotReward, c.wantReward) {
			t.Fatalf("test case %d: %s, the coinbase reward got: %v, want: %v", i, c.desc, *gotReward, c.wantReward)
		}
	}
}

func TestCountCoinbaseTxRewards(t *testing.T) {
	consensus.ActiveNetParams.ProducerSubsidys = []consensus.ProducerSubsidy{
		{BeginBlock: 0, EndBlock: 0, Subsidy: 24},
		{BeginBlock: 1, EndBlock: 840000, Subsidy: 24},
		{BeginBlock: 840001, EndBlock: 1680000, Subsidy: 12},
		{BeginBlock: 1680001, EndBlock: 3360000, Subsidy: 6},
	}

	cases := []struct {
		desc                string
		block               *types.Block
		consensusResult     *state.ConsensusResult
		wantRewards         []state.CoinbaseReward
		wantConsensusResult *state.ConsensusResult
	}{
		{
			desc: "the block height is RoundVoteBlockNums - 1",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height: consensus.ActiveNetParams.RoundVoteBlockNums - 1,
				},
				Transactions: []*types.Tx{nil},
			},
			consensusResult: &state.ConsensusResult{
				CoinbaseReward: map[string]uint64{
					"51": 10,
				},
			},
			wantRewards: []state.CoinbaseReward{},
			wantConsensusResult: &state.ConsensusResult{
				CoinbaseReward: map[string]uint64{
					"51": 34,
				},
			},
		},
		{
			desc: "the block height is RoundVoteBlockNums",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height: consensus.ActiveNetParams.RoundVoteBlockNums,
				},
				Transactions: []*types.Tx{nil},
			},
			consensusResult: &state.ConsensusResult{
				CoinbaseReward: map[string]uint64{
					"51": 10,
					"52": 20,
				},
			},
			wantRewards: []state.CoinbaseReward{},
			wantConsensusResult: &state.ConsensusResult{
				CoinbaseReward: map[string]uint64{
					"51": 34,
					"52": 20,
				},
			},
		},
		{
			desc: "the block height is RoundVoteBlockNums + 1",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height: consensus.ActiveNetParams.RoundVoteBlockNums + 1,
				},
				Transactions: []*types.Tx{nil},
			},
			consensusResult: &state.ConsensusResult{
				CoinbaseReward: map[string]uint64{
					"51": 10,
				},
			},
			wantRewards: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         10,
					ControlProgram: []byte{0x51},
				},
			},
			wantConsensusResult: &state.ConsensusResult{
				CoinbaseReward: map[string]uint64{
					"51": 24,
				},
			},
		},
		{
			desc: "the block height is RoundVoteBlockNums * 2",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height: consensus.ActiveNetParams.RoundVoteBlockNums * 2,
				},
				Transactions: []*types.Tx{nil},
			},
			consensusResult: &state.ConsensusResult{
				CoinbaseReward: map[string]uint64{
					"51": 10,
				},
			},
			wantRewards: []state.CoinbaseReward{},
			wantConsensusResult: &state.ConsensusResult{
				CoinbaseReward: map[string]uint64{
					"51": 34,
				},
			},
		},
		{
			desc: "the block height is 2*RoundVoteBlockNums + 1",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height: 2*consensus.ActiveNetParams.RoundVoteBlockNums + 1,
				},
				Transactions: []*types.Tx{nil},
			},
			consensusResult: &state.ConsensusResult{
				CoinbaseReward: map[string]uint64{
					"51": 10,
					"52": 20,
				},
			},
			wantRewards: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         20,
					ControlProgram: []byte{0x52},
				},
				state.CoinbaseReward{
					Amount:         10,
					ControlProgram: []byte{0x51},
				},
			},
			wantConsensusResult: &state.ConsensusResult{
				CoinbaseReward: map[string]uint64{
					"51": 24,
				},
			},
		},
	}

	for i, c := range cases {
		rewards, err := c.consensusResult.GetCoinbaseRewards(c.block.Height - 1)
		if err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(rewards, c.wantRewards) {
			t.Fatalf("test case %d: %s, the coinbase reward got: %v, want: %v", i, c.desc, rewards, c.wantRewards)
		}

		// create coinbase transaction
		c.block.Transactions[0], err = createCoinbaseTxByReward(nil, c.block.Height, rewards)
		if err != nil {
			t.Fatal(err)
		}

		if err := c.consensusResult.AttachCoinbaseReward(c.block); err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(c.consensusResult, c.wantConsensusResult) {
			t.Fatalf("test case %d: %s, the consensusResult got: %v, want: %v", i, c.desc, c.consensusResult, c.wantConsensusResult)
		}
	}
}
