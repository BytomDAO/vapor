package proposal

import (
	"testing"

	"github.com/vapor/consensus"
	"github.com/vapor/protocol/state"
	"github.com/vapor/testutil"
)

func TestCreateCoinbaseTx(t *testing.T) {
	reductionInterval := uint64(840000)
	cases := []struct {
		desc            string
		consensusResult *state.ConsensusResult
		txFee           uint64
		wantOutputs     []state.CoinbaseReward
	}{
		{
			desc: "the coinbase block height is reductionInterval",
			consensusResult: &state.ConsensusResult{
				BlockHeight: reductionInterval - 1,
			},
			txFee: 100000000,
			wantOutputs: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         uint64(0),
					ControlProgram: []byte{0x51},
				},
				state.CoinbaseReward{
					Amount:         uint64(100000000),
					ControlProgram: []byte{0x51},
				},
			},
		},
		{
			desc: "the coinbase block height is consensus.RoundVoteBlockNums",
			consensusResult: &state.ConsensusResult{
				BlockHeight: consensus.RoundVoteBlockNums - 1,
			},
			txFee: 200000000,
			wantOutputs: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         uint64(0),
					ControlProgram: []byte{0x51},
				},
				state.CoinbaseReward{
					Amount:         uint64(200000000),
					ControlProgram: []byte{0x51},
				},
			},
		},
		{
			desc: "the coinbase block height is 2*consensus.RoundVoteBlockNums",
			consensusResult: &state.ConsensusResult{
				BlockHeight: 2*consensus.RoundVoteBlockNums - 1,
			},
			txFee: 300000000,
			wantOutputs: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         uint64(0),
					ControlProgram: []byte{0x51},
				},
				state.CoinbaseReward{
					Amount:         uint64(300000000),
					ControlProgram: []byte{0x51},
				},
			},
		},
		{
			desc: "the coinbase block height with multi outputs",
			consensusResult: &state.ConsensusResult{
				BlockHeight: reductionInterval - 1,
				RewardOfCoinbase: map[string]uint64{
					"51": 100,
					"52": 200,
					"55": 500,
					"53": 300,
				},
			},
			txFee: 2000,
			wantOutputs: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         uint64(0),
					ControlProgram: []byte{0x51},
				},
				state.CoinbaseReward{
					Amount:         uint64(200),
					ControlProgram: []byte{0x52},
				},
				state.CoinbaseReward{
					Amount:         uint64(300),
					ControlProgram: []byte{0x53},
				},
				state.CoinbaseReward{
					Amount:         uint64(500),
					ControlProgram: []byte{0x55},
				},
				state.CoinbaseReward{
					Amount:         uint64(2100),
					ControlProgram: []byte{0x51},
				},
			},
		},
		{
			desc: "the coinbase block height is reductionInterval - 1",
			consensusResult: &state.ConsensusResult{
				BlockHeight: reductionInterval - 2,
			},
			txFee: 100000000,
			wantOutputs: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         uint64(0),
					ControlProgram: []byte{0x51},
				},
			},
		},
		{
			desc: "the coinbase block height is reductionInterval + 1",
			consensusResult: &state.ConsensusResult{
				BlockHeight: reductionInterval,
			},
			txFee: 0,
			wantOutputs: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         uint64(0),
					ControlProgram: []byte{0x51},
				},
			},
		},
		{
			desc: "the coinbase block height is reductionInterval * 2",
			consensusResult: &state.ConsensusResult{
				BlockHeight: 2*reductionInterval - 1,
			},
			txFee: 100000000,
			wantOutputs: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         uint64(0),
					ControlProgram: []byte{0x51},
				},
				state.CoinbaseReward{
					Amount:         uint64(100000000),
					ControlProgram: []byte{0x51},
				},
			},
		},
	}

	for i, c := range cases {
		coinbaseTx, err := createCoinbaseTx(c.consensusResult, nil, c.txFee)
		if err != nil {
			t.Fatal(err)
		}

		gotOutputs := []state.CoinbaseReward{}
		for _, output := range coinbaseTx.Outputs {
			gotOutputs = append(gotOutputs, state.CoinbaseReward{
				Amount:         output.AssetAmount().Amount,
				ControlProgram: output.ControlProgram(),
			})
		}

		if ok := testutil.DeepEqual(gotOutputs, c.wantOutputs); !ok {
			t.Fatalf("coinbase tx reward dismatch, case: %d, got: %d, want: %d", i, gotOutputs, c.wantOutputs)
		}
	}
}
