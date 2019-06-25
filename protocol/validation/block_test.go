package validation

import (
	"math"
	"testing"
	"time"

	"github.com/vapor/consensus"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm"
	"github.com/vapor/protocol/vm/vmutil"
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
			desc:       "timestamp less than past median time (blocktest#1005)",
			blockTime:  1520005500,
			parentTime: []uint64{1520000000, 1520000500, 1520001000, 1520001500, 1520002000, 1520002500, 1520003000, 1520003500, 1520004000, 1520004500, 1520005000},
			err:        nil,
		},
		{
			desc:       "timestamp greater than max limit (blocktest#1006)",
			blockTime:  99999999990000,
			parentTime: []uint64{15200000000000},
			err:        errBadTimestamp,
		},
		{
			desc:       "timestamp of the block and the parent block are both greater than max limit (blocktest#1007)",
			blockTime:  uint64(time.Now().UnixNano()/int64(time.Millisecond)) + consensus.MaxTimeOffsetMs + 2000,
			parentTime: []uint64{uint64(time.Now().UnixNano()/int64(time.Millisecond)) + consensus.MaxTimeOffsetMs + 1000},
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

func TestCheckCoinbaseAmount(t *testing.T) {
	cases := []struct {
		txs    []*types.Tx
		amount uint64
		err    error
	}{
		{
			txs: []*types.Tx{
				types.NewTx(types.TxData{
					Inputs:  []*types.TxInput{types.NewCoinbaseInput(nil)},
					Outputs: []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 5000, nil)},
				}),
			},
			amount: 5000,
			err:    nil,
		},
		{
			txs: []*types.Tx{
				types.NewTx(types.TxData{
					Inputs:  []*types.TxInput{types.NewCoinbaseInput(nil)},
					Outputs: []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 5000, nil)},
				}),
			},
			amount: 6000,
			err:    ErrWrongCoinbaseTransaction,
		},
		{
			txs:    []*types.Tx{},
			amount: 5000,
			err:    ErrWrongCoinbaseTransaction,
		},
	}

	block := new(types.Block)
	for i, c := range cases {
		block.Transactions = c.txs
		if err := checkCoinbaseAmount(types.MapBlock(block), c.amount); rootErr(err) != c.err {
			t.Errorf("case %d got error %s, want %s", i, err, c.err)
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
			block: &bc.Block{BlockHeader: &bc.BlockHeader{
				Version: 2,
			}},
			parent: &types.BlockHeader{
				Version: 1,
			},
			err: errVersionRegression,
		},
		{
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
			desc: "the prev block hash not equals to the hash of parent (blocktest#1004)",
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
			desc: "version greater than 1 (blocktest#1001)",
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
			desc: "version equals 0 (blocktest#1002)",
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
			desc: "version equals max uint64 (blocktest#1003)",
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

// TestValidateBlock test the ValidateBlock function
func TestValidateBlock(t *testing.T) {
	cp, _ := vmutil.DefaultCoinbaseProgram()
	parent := &types.BlockHeader{
		Version:           1,
		Height:            0,
		Timestamp:         1523352600000,
		PreviousBlockHash: bc.Hash{V0: 0},
	}
	parentHash := parent.Hash()

	cases := []struct {
		desc   string
		block  *bc.Block
		parent *types.BlockHeader
		err    error
	}{
		{
			desc: "The calculated transaction merkel root hash is not equals to the hash of the block header (blocktest#1009)",
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
						Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, consensus.BlockSubsidy(0), cp)},
					}),
				},
			},
			parent: parent,
			err:    errMismatchedMerkleRoot,
		},
		{
			desc: "The calculated transaction status merkel root hash is not equals to the hash of the block header (blocktest#1009)",
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
						Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, consensus.BlockSubsidy(0), cp)},
					}),
				},
			},
			parent: parent,
			err:    errMismatchedMerkleRoot,
		},
		{
			desc: "the coinbase amount is less than the real coinbase amount (txtest#1014)",
			block: &bc.Block{
				ID: bc.Hash{V0: 1},
				BlockHeader: &bc.BlockHeader{
					Version:         1,
					Height:          1,
					Timestamp:       1523352601000,
					PreviousBlockId: &parentHash,
				},
				Transactions: []*bc.Tx{
					types.MapTx(&types.TxData{
						Version:        1,
						SerializedSize: 1,
						Inputs:         []*types.TxInput{types.NewCoinbaseInput(nil)},
						Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 41250000000, cp)},
					}),
					types.MapTx(&types.TxData{
						Version:        1,
						SerializedSize: 1,
						Inputs:         []*types.TxInput{types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 100000000, 0, cp)},
						Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 90000000, cp)},
					}),
				},
			},
			parent: parent,
			err:    ErrWrongCoinbaseTransaction,
		},
	}

	for i, c := range cases {
		err := ValidateBlock(c.block, c.parent)
		if rootErr(err) != c.err {
			t.Errorf("case #%d (%s) got error %s, want %s", i, c.desc, err, c.err)
		}
	}
}

// TestGasOverBlockLimit check if the gas of the block has the max limit (blocktest#1012)
func TestGasOverBlockLimit(t *testing.T) {
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
				Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 41250000000, cp)},
			}),
		},
	}

	for i := 0; i < 100; i++ {
		block.Transactions = append(block.Transactions, types.MapTx(&types.TxData{
			Version:        1,
			SerializedSize: 100000,
			Inputs: []*types.TxInput{
				types.NewSpendInput([][]byte{}, *newHash(8), *consensus.BTMAssetID, 10000000000, 0, cp),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(*consensus.BTMAssetID, 9000000000, cp),
			},
		}))
	}

	if err := ValidateBlock(block, parent); err != errOverBlockLimit {
		t.Errorf("got error %s, want %s", err, errOverBlockLimit)
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
			TransactionsRoot:      &bc.Hash{V0: 12212572290317752069, V1: 8979003395977198825, V2: 3978010681554327084, V3: 12322462500143540195},
			TransactionStatusHash: &bc.Hash{V0: 8682965660674182538, V1: 8424137560837623409, V2: 6979974817894224946, V3: 4673809519342015041},
		},
		Transactions: []*bc.Tx{
			types.MapTx(&types.TxData{
				Version:        1,
				SerializedSize: 1,
				Inputs:         []*types.TxInput{types.NewCoinbaseInput(nil)},
				Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 199998224, cp)},
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

	if err := ValidateBlock(block, parent); err != nil {
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
