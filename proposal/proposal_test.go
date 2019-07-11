package proposal

import (
	"encoding/hex"
	"testing"

	"github.com/vapor/consensus"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
	"github.com/vapor/protocol/vm/vmutil"
	"github.com/vapor/testutil"
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
		desc            string
		block           *types.Block
		consensusResult *state.ConsensusResult
		wantReward      state.CoinbaseReward
	}{
		{
			desc: "the block height is reductionInterval - 1",
			block: &types.Block{
				BlockHeader: types.BlockHeader{
					Height: reductionInterval - 1,
				},
				Transactions: []*types.Tx{nil},
			},
			consensusResult: &state.ConsensusResult{
				CoinbaseReward: map[string]uint64{},
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
				Transactions: []*types.Tx{nil},
			},
			consensusResult: &state.ConsensusResult{
				CoinbaseReward: map[string]uint64{},
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
				Transactions: []*types.Tx{nil},
			},
			consensusResult: &state.ConsensusResult{
				CoinbaseReward: map[string]uint64{},
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
				Transactions: []*types.Tx{nil},
			},
			consensusResult: &state.ConsensusResult{
				CoinbaseReward: map[string]uint64{},
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
				Transactions: []*types.Tx{nil},
			},
			consensusResult: &state.ConsensusResult{
				CoinbaseReward: map[string]uint64{},
			},
			wantReward: state.CoinbaseReward{
				Amount:         6,
				ControlProgram: []byte{0x51},
			},
		},
	}

	var err error
	for i, c := range cases {
		c.block.Transactions[0], err = createCoinbaseTx(nil, c.block.Height)
		if err != nil {
			t.Fatal(err)
		}

		if err := c.consensusResult.AttachCoinbaseReward(c.block); err != nil {
			t.Fatal(err)
		}

		defaultProgram, _ := vmutil.DefaultCoinbaseProgram()
		gotReward := state.CoinbaseReward{
			Amount:         c.consensusResult.CoinbaseReward[hex.EncodeToString(defaultProgram)],
			ControlProgram: defaultProgram,
		}

		if !testutil.DeepEqual(gotReward, c.wantReward) {
			t.Fatalf("test case %d: %s, the coinbase reward got: %v, want: %v", i, c.desc, gotReward, c.wantReward)
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
		desc            string
		block           *types.Block
		consensusResult *state.ConsensusResult
		wantRewards     []state.CoinbaseReward
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
			wantRewards: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         20,
					ControlProgram: []byte{0x52},
				},
				state.CoinbaseReward{
					Amount:         34,
					ControlProgram: []byte{0x51},
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
			wantRewards: []state.CoinbaseReward{},
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
					"52": 20,
					"53": 30,
				},
			},
			wantRewards: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         20,
					ControlProgram: []byte{0x52},
				},
				state.CoinbaseReward{
					Amount:         30,
					ControlProgram: []byte{0x53},
				},
				state.CoinbaseReward{
					Amount:         34,
					ControlProgram: []byte{0x51},
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
				CoinbaseReward: map[string]uint64{},
			},
			wantRewards: []state.CoinbaseReward{},
		},
	}

	var err error
	for i, c := range cases {
		c.block.Transactions[0], err = createCoinbaseTx(nil, c.block.Height)
		if err != nil {
			t.Fatal(err)
		}

		if err := c.consensusResult.AttachCoinbaseReward(c.block); err != nil {
			t.Fatal(err)
		}

		rewards, err := c.consensusResult.GetCoinbaseRewards(c.block.Height)
		if err != nil {
			t.Fatal(err)
		}
		if !testutil.DeepEqual(rewards, c.wantRewards) {
			t.Fatalf("test case %d: %s, the coinbase reward got: %v, want: %v", i, c.desc, rewards, c.wantRewards)
		}
	}
}
