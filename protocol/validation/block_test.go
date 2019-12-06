package validation

import (
	"math"
	"testing"
	"time"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
	"github.com/bytom/vapor/protocol/vm"
	"github.com/bytom/vapor/protocol/vm/vmutil"
	"github.com/bytom/vapor/testutil"
)

func TestCheckBlockTime(t *testing.T) {
	cases := []struct {
		desc       string
		blockTime  uint64
		parentTime []uint64
		err        error
	}{
		{
			blockTime:  1520000500,
			parentTime: []uint64{1520000000},
			err:        nil,
		},
		{
			desc:       "timestamp less than past median time",
			blockTime:  1520005500,
			parentTime: []uint64{1520000000, 1520000500, 1520001000, 1520001500, 1520002000, 1520002500, 1520003000, 1520003500, 1520004000, 1520004500, 1520005000},
			err:        nil,
		},
		{
			desc:       "timestamp greater than max limit",
			blockTime:  99999999990000,
			parentTime: []uint64{15200000000000},
			err:        errBadTimestamp,
		},
		{
			desc:       "timestamp of the block and the parent block are both greater than max limit",
			blockTime:  uint64(time.Now().UnixNano()/int64(time.Millisecond)) + consensus.ActiveNetParams.MaxTimeOffsetMs + 2000,
			parentTime: []uint64{uint64(time.Now().UnixNano()/int64(time.Millisecond)) + consensus.ActiveNetParams.MaxTimeOffsetMs + 1000},
			err:        errBadTimestamp,
		},
	}

	parent := &types.BlockHeader{Version: 1}
	block := &bc.Block{
		BlockHeader: &bc.BlockHeader{Version: 1},
	}

	for i, c := range cases {
		parent.Timestamp = c.parentTime[0]
		parentSuccessor := parent
		for i := 1; i < len(c.parentTime); i++ {
			Previous := &types.BlockHeader{Version: 1, Timestamp: c.parentTime[i]}
			parentSuccessor.PreviousBlockHash = Previous.Hash()
			parentSuccessor = Previous
		}

		block.Timestamp = c.blockTime
		if err := checkBlockTime(block, parent); rootErr(err) != c.err {
			t.Errorf("case %d got error %s, want %s", i, err, c.err)
		}
	}
}

func TestCheckCoinbaseTx(t *testing.T) {
	cases := []struct {
		desc    string
		txs     []*types.Tx
		rewards []state.CoinbaseReward
		err     error
	}{
		{
			desc: "zero coinbase amount",
			txs: []*types.Tx{
				types.NewTx(types.TxData{
					Inputs:  []*types.TxInput{types.NewCoinbaseInput(nil)},
					Outputs: []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 0, []byte{0x51})},
				}),
			},
			rewards: []state.CoinbaseReward{},
			err:     nil,
		},
		{
			desc: "zero coinbase amount and aggregate rewards",
			txs: []*types.Tx{
				types.NewTx(types.TxData{
					Inputs: []*types.TxInput{types.NewCoinbaseInput(nil)},
					Outputs: []*types.TxOutput{
						types.NewIntraChainOutput(*consensus.BTMAssetID, 0, []byte{0x51}),
						types.NewIntraChainOutput(*consensus.BTMAssetID, 20000, []byte{0x51}),
						types.NewIntraChainOutput(*consensus.BTMAssetID, 10000, []byte{0x52}),
					},
				}),
			},
			rewards: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         20000,
					ControlProgram: []byte{0x51},
				},
				state.CoinbaseReward{
					Amount:         10000,
					ControlProgram: []byte{0x52},
				},
			},
			err: nil,
		},
		{
			desc:    "wrong coinbase transaction with block is empty",
			txs:     []*types.Tx{},
			rewards: []state.CoinbaseReward{},
			err:     ErrWrongCoinbaseTransaction,
		},
		{
			desc: "wrong coinbase transaction with dismatch number of outputs",
			txs: []*types.Tx{
				types.NewTx(types.TxData{
					Inputs: []*types.TxInput{types.NewCoinbaseInput(nil)},
					Outputs: []*types.TxOutput{
						types.NewIntraChainOutput(*consensus.BTMAssetID, 0, []byte{0x51}),
						types.NewIntraChainOutput(*consensus.BTMAssetID, 20000, []byte{0x51}),
						types.NewIntraChainOutput(*consensus.BTMAssetID, 10000, []byte{0x52}),
					},
				}),
			},
			rewards: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         20000,
					ControlProgram: []byte{0x51},
				},
			},
			err: ErrWrongCoinbaseTransaction,
		},
		{
			desc: "wrong coinbase transaction with dismatch output amount",
			txs: []*types.Tx{
				types.NewTx(types.TxData{
					Inputs: []*types.TxInput{types.NewCoinbaseInput(nil)},
					Outputs: []*types.TxOutput{
						types.NewIntraChainOutput(*consensus.BTMAssetID, 0, []byte{0x51}),
						types.NewIntraChainOutput(*consensus.BTMAssetID, 20000, []byte{0x51}),
						types.NewIntraChainOutput(*consensus.BTMAssetID, 10000, []byte{0x52}),
					},
				}),
			},
			rewards: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         10000,
					ControlProgram: []byte{0x51},
				},
				state.CoinbaseReward{
					Amount:         10000,
					ControlProgram: []byte{0x52},
				},
			},
			err: ErrWrongCoinbaseTransaction,
		},
		{
			desc: "wrong coinbase transaction with dismatch output control_program",
			txs: []*types.Tx{
				types.NewTx(types.TxData{
					Inputs: []*types.TxInput{types.NewCoinbaseInput(nil)},
					Outputs: []*types.TxOutput{
						types.NewIntraChainOutput(*consensus.BTMAssetID, 0, []byte{0x51}),
						types.NewIntraChainOutput(*consensus.BTMAssetID, 20000, []byte{0x51}),
						types.NewIntraChainOutput(*consensus.BTMAssetID, 10000, []byte{0x52}),
					},
				}),
			},
			rewards: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         20000,
					ControlProgram: []byte{0x51},
				},
				state.CoinbaseReward{
					Amount:         10000,
					ControlProgram: []byte{0x53},
				},
			},
			err: ErrWrongCoinbaseTransaction,
		},
	}

	block := new(types.Block)
	for i, c := range cases {
		block.Transactions = c.txs
		if err := checkCoinbaseTx(types.MapBlock(block), c.rewards); rootErr(err) != c.err {
			t.Errorf("case %d got error %s, want %T", i, err, c.err)
		}
	}
}

func TestValidateBlockHeader(t *testing.T) {
	parent := &types.BlockHeader{
		Version:   1,
		Height:    0,
		Timestamp: 1523352600000,
	}
	parentHash := parent.Hash()

	cases := []struct {
		desc   string
		block  *bc.Block
		parent *types.BlockHeader
		err    error
	}{
		{
			desc: "dismatch version",
			block: &bc.Block{BlockHeader: &bc.BlockHeader{
				Version: 2,
			}},
			parent: &types.BlockHeader{
				Version: 1,
			},
			err: errVersionRegression,
		},
		{
			desc: "misordered block height",
			block: &bc.Block{BlockHeader: &bc.BlockHeader{
				Version: 1,
				Height:  20,
			}},
			parent: &types.BlockHeader{
				Version: 1,
				Height:  18,
			},
			err: errMisorderedBlockHeight,
		},
		{
			desc: "the prev block hash not equals to the hash of parent",
			block: &bc.Block{BlockHeader: &bc.BlockHeader{
				Version:         1,
				Height:          20,
				PreviousBlockId: &bc.Hash{V0: 20},
			}},
			parent: &types.BlockHeader{
				Version:           1,
				Height:            19,
				PreviousBlockHash: bc.Hash{V0: 19},
			},
			err: errMismatchedBlock,
		},
		{
			desc: "normal block",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version:         1,
					Height:          1,
					Timestamp:       1523352601000,
					PreviousBlockId: &parentHash,
				},
			},
			parent: parent,
			err:    nil,
		},
		{
			desc: "version greater than 1",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version: 2,
				},
			},
			parent: &types.BlockHeader{
				Version: 1,
			},
			err: errVersionRegression,
		},
		{
			desc: "version equals 0",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version: 0,
				},
			},
			parent: &types.BlockHeader{
				Version: 1,
			},
			err: errVersionRegression,
		},
		{
			desc: "version equals max uint64",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version: math.MaxUint64,
				},
			},
			parent: &types.BlockHeader{
				Version: 1,
			},
			err: errVersionRegression,
		},
	}

	for i, c := range cases {
		if err := ValidateBlockHeader(c.block, c.parent); rootErr(err) != c.err {
			t.Errorf("case %d (%s) got error %s, want %s", i, c.desc, err, c.err)
		}
	}
}

func TestValidateBlock(t *testing.T) {
	cp, _ := vmutil.DefaultCoinbaseProgram()
	parent := &types.BlockHeader{
		Version:           1,
		Height:            0,
		Timestamp:         1523352600000,
		PreviousBlockHash: bc.Hash{V0: 0},
	}
	parentHash := parent.Hash()
	txsRoot := testutil.MustDecodeHash("001e21b9618c503d909c1e0b32bab9ccf80c538b35d49ac7fffcef98eb373b23")
	txStatusHash := testutil.MustDecodeHash("6978a65b4ee5b6f4914fe5c05000459a803ecf59132604e5d334d64249c5e50a")

	txs := []*bc.Tx{
		types.MapTx(&types.TxData{
			Version:        1,
			SerializedSize: 1,
			Inputs:         []*types.TxInput{types.NewCoinbaseInput(nil)},
			Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 0, cp)},
		}),
	}

	for i := 1; i <= 100; i++ {
		txs = append(txs, types.MapTx(&types.TxData{
			Version:        1,
			SerializedSize: 100000,
			Inputs:         []*types.TxInput{types.NewSpendInput([][]byte{}, bc.Hash{V0: uint64(i)}, *consensus.BTMAssetID, 10000000000, 0, cp)},
			Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 9000000000, cp)},
		}))
	}

	cases := []struct {
		desc    string
		block   *bc.Block
		parent  *types.BlockHeader
		rewards []state.CoinbaseReward
		err     error
	}{
		{
			desc: "validate transactions with output amount great than input amount",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version:               1,
					Height:                1,
					Timestamp:             1523352601000,
					PreviousBlockId:       &parentHash,
					TransactionsRoot:      &bc.Hash{V0: 16229071813194843118, V1: 7413717724217377663, V2: 10255217553502780716, V3: 17975900656333257644},
					TransactionStatusHash: &txStatusHash,
				},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						Version:        1,
						SerializedSize: 1,
						Inputs:         []*types.TxInput{types.NewCoinbaseInput(nil)},
						Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 0, cp)},
					}),
					types.MapTx(&types.TxData{
						Version:        1,
						SerializedSize: 528,
						Inputs:         []*types.TxInput{types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp)},
						Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 100000000, cp)},
					}),
					types.MapTx(&types.TxData{
						Version:        1,
						SerializedSize: 528,
						Inputs:         []*types.TxInput{types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp)},
						Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 200000000, cp)},
					}),
				},
			},
			parent: parent,
			err:    ErrGasCalculate,
		},
		{
			desc: "validate block with the total transations used gas is over than the limit",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version:               1,
					Height:                1,
					Timestamp:             1523352601000,
					PreviousBlockId:       &parentHash,
					TransactionsRoot:      &bc.Hash{V0: 11799591616144015196, V1: 10485585098288308103, V2: 9819002243760462505, V3: 10203115105872271656},
					TransactionStatusHash: &txStatusHash,
				},
				Transactions: txs,
			},
			parent: parent,
			err:    errOverBlockLimit,
		},
		{
			desc: "The calculated transaction merkel root hash is not equals to the hash of the block header",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version:          1,
					Height:           1,
					Timestamp:        1523352601000,
					PreviousBlockId:  &parentHash,
					TransactionsRoot: &bc.Hash{V0: 1},
				},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						Version:        1,
						SerializedSize: 1,
						Inputs:         []*types.TxInput{types.NewCoinbaseInput(nil)},
						Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 0, cp)},
					}),
				},
			},
			parent: parent,
			err:    errMismatchedMerkleRoot,
		},
		{
			desc: "The calculated transaction status merkel root hash is not equals to the hash of the block header",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version:               1,
					Height:                1,
					Timestamp:             1523352601000,
					PreviousBlockId:       &parentHash,
					TransactionsRoot:      &bc.Hash{V0: 6294987741126419124, V1: 12520373106916389157, V2: 5040806596198303681, V3: 1151748423853876189},
					TransactionStatusHash: &bc.Hash{V0: 1},
				},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						Version:        1,
						SerializedSize: 1,
						Inputs:         []*types.TxInput{types.NewCoinbaseInput(nil)},
						Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 0, cp)},
					}),
				},
			},
			parent: parent,
			err:    errMismatchedMerkleRoot,
		},
		{
			desc: "the coinbase amount is not equal to the real coinbase outputs",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version:               1,
					Height:                1,
					Timestamp:             1523352601000,
					PreviousBlockId:       &parentHash,
					TransactionsRoot:      &txsRoot,
					TransactionStatusHash: &txStatusHash,
				},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						Version:        1,
						SerializedSize: 1,
						Inputs:         []*types.TxInput{types.NewCoinbaseInput(nil)},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(*consensus.BTMAssetID, 0, cp),
							types.NewIntraChainOutput(*consensus.BTMAssetID, 20000, []byte{0x51}),
						},
					}),
				},
			},
			parent: parent,
			rewards: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         20000,
					ControlProgram: []byte{0x51},
				},
				state.CoinbaseReward{
					Amount:         10000,
					ControlProgram: []byte{0x52},
				},
			},
			err: ErrWrongCoinbaseTransaction,
		},
		{
			desc: "the coinbase program is not equal to the real coinbase outputs",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version:               1,
					Height:                1,
					Timestamp:             1523352601000,
					PreviousBlockId:       &parentHash,
					TransactionsRoot:      &txsRoot,
					TransactionStatusHash: &txStatusHash,
				},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						Version:        1,
						SerializedSize: 1,
						Inputs:         []*types.TxInput{types.NewCoinbaseInput(nil)},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(*consensus.BTMAssetID, 0, cp),
							types.NewIntraChainOutput(*consensus.BTMAssetID, 20000, []byte{0x51}),
							types.NewIntraChainOutput(*consensus.BTMAssetID, 10000, []byte{0x61}),
						},
					}),
				},
			},
			parent: parent,
			rewards: []state.CoinbaseReward{
				state.CoinbaseReward{
					Amount:         20000,
					ControlProgram: []byte{0x51},
				},
				state.CoinbaseReward{
					Amount:         10000,
					ControlProgram: []byte{0x52},
				},
			},
			err: ErrWrongCoinbaseTransaction,
		},
		{
			desc: "the coinbase amount is equal to the real coinbase amount",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version:               1,
					Height:                1,
					Timestamp:             1523352601000,
					PreviousBlockId:       &parentHash,
					TransactionsRoot:      &bc.Hash{V0: 16229071813194843118, V1: 7413717724217377663, V2: 10255217553502780716, V3: 17975900656333257644},
					TransactionStatusHash: &txStatusHash,
				},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						Version:        1,
						SerializedSize: 1,
						Inputs:         []*types.TxInput{types.NewCoinbaseInput(nil)},
						Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 0, cp)},
					}),
					types.MapTx(&types.TxData{
						Version:        1,
						SerializedSize: 1,
						Inputs:         []*types.TxInput{types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp)},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(*consensus.BTMAssetID, 0, cp),
							types.NewIntraChainOutput(*consensus.BTMAssetID, 100000000, cp),
						},
					}),
				},
			},
			parent: parent,
			err:    nil,
		},
	}

	for i, c := range cases {
		err := ValidateBlock(c.block, c.parent, c.rewards)
		if rootErr(err) != c.err {
			t.Errorf("case #%d (%s) got error %s, want %s", i, c.desc, err, c.err)
		}
	}
}

// TestSetTransactionStatus verify the transaction status is set correctly (blocktest#1010)
func TestSetTransactionStatus(t *testing.T) {
	cp, _ := vmutil.DefaultCoinbaseProgram()
	parent := &types.BlockHeader{
		Version:           1,
		Height:            0,
		Timestamp:         1523352600000,
		PreviousBlockHash: bc.Hash{V0: 0},
	}
	parentHash := parent.Hash()

	block := &bc.Block{
		ID: bc.Hash{V0: 1},
		BlockHeader: &bc.BlockHeader{
			Version:               1,
			Height:                1,
			Timestamp:             1523352601000,
			PreviousBlockId:       &parentHash,
			TransactionsRoot:      &bc.Hash{V0: 8176741810667217458, V1: 14830712230021600370, V2: 8921661778795432162, V3: 3391855546006364086},
			TransactionStatusHash: &bc.Hash{V0: 8682965660674182538, V1: 8424137560837623409, V2: 6979974817894224946, V3: 4673809519342015041},
		},
		Transactions: []*bc.Tx{
			types.MapTx(&types.TxData{
				Version:        1,
				SerializedSize: 1,
				Inputs:         []*types.TxInput{types.NewCoinbaseInput(nil)},
				Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 0, cp)},
			}),
			types.MapTx(&types.TxData{
				Version:        1,
				SerializedSize: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp),
					types.NewSpendInput([][]byte{}, *newHash(8), bc.AssetID{V0: 1}, 1000, 0, []byte{byte(vm.OP_FALSE)}),
				},
				Outputs: []*types.TxOutput{
					types.NewIntraChainOutput(*consensus.BTMAssetID, 888, cp),
					types.NewIntraChainOutput(bc.AssetID{V0: 1}, 1000, cp),
				},
			}),
			types.MapTx(&types.TxData{
				Version:        1,
				SerializedSize: 1,
				Inputs: []*types.TxInput{
					types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp),
				},
				Outputs: []*types.TxOutput{
					types.NewIntraChainOutput(*consensus.BTMAssetID, 888, cp),
				},
			}),
		},
	}

	if err := ValidateBlock(block, parent, []state.CoinbaseReward{}); err != nil {
		t.Fatal(err)
	}

	expectTxStatuses := []bool{false, true, false}
	txStatuses := block.GetTransactionStatus().VerifyStatus
	if len(expectTxStatuses) != len(txStatuses) {
		t.Error("the size of expect tx status is not equals to size of got tx status")
	}

	for i, status := range txStatuses {
		if expectTxStatuses[i] != status.StatusFail {
			t.Errorf("got tx status: %v, expect tx status: %v\n", status.StatusFail, expectTxStatuses[i])
		}
	}
}
