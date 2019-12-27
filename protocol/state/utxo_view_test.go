package state

import (
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/database/storage"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/testutil"
)

var defaultEntry = map[bc.Hash]bc.Entry{
	bc.Hash{V0: 0}: &bc.IntraChainOutput{
		Source: &bc.ValueSource{
			Value: &bc.AssetAmount{
				AssetId: &bc.AssetID{V0: 0},
			},
		},
	},
}

var coinbaseEntry = map[bc.Hash]bc.Entry{
	bc.Hash{V0: 0}: &bc.IntraChainOutput{
		Source: &bc.ValueSource{
			Value: &bc.AssetAmount{
				AssetId: consensus.BTMAssetID,
			},
		},
	},
	bc.Hash{V0: 1}: &bc.IntraChainOutput{
		Source: &bc.ValueSource{
			Value: &bc.AssetAmount{
				AssetId: consensus.BTMAssetID,
				Amount:  uint64(100),
			},
		},
	},
}

var voteEntry = map[bc.Hash]bc.Entry{
	bc.Hash{V0: 0}: &bc.VoteOutput{
		Source: &bc.ValueSource{
			Value: &bc.AssetAmount{
				AssetId: &bc.AssetID{V0: 0},
			},
		},
	},
}

var gasOnlyTxEntry = map[bc.Hash]bc.Entry{
	bc.Hash{V1: 0}: &bc.IntraChainOutput{
		Source: &bc.ValueSource{
			Value: &bc.AssetAmount{
				AssetId: consensus.BTMAssetID,
			},
		},
	},
	bc.Hash{V1: 1}: &bc.IntraChainOutput{
		Source: &bc.ValueSource{
			Value: &bc.AssetAmount{
				AssetId: &bc.AssetID{V0: 999},
			},
		},
	},
}

func TestApplyBlock(t *testing.T) {
	cases := []struct {
		block     *bc.Block
		inputView *UtxoViewpoint
		fetchView *UtxoViewpoint
		gasOnlyTx bool
		err       bool
	}{
		{
			// can't find prevout in tx entries
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 1},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
				},
			},
			fetchView: NewUtxoViewpoint(),
			err:       true,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: NewUtxoViewpoint(),
			fetchView: NewUtxoViewpoint(),
			err:       true,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, true),
				},
			},
			err: true,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
				},
			},
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, true),
				},
			},
			err: false,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					Height:            consensus.MainNetParams.CoinbasePendingBlockNumber + 1,
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.CoinbaseUTXOType, 0, false),
				},
			},
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.CoinbaseUTXOType, 0, true),
				},
			},
			err: false,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					Height:            0,
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.CoinbaseUTXOType, 0, false),
				},
			},
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.CoinbaseUTXOType, 0, true),
				},
			},
			err: true,
		},
		{
			// output will be store
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{
								&bc.Hash{V0: 0},
								&bc.Hash{V0: 1},
							},
						},
						SpentOutputIDs: []bc.Hash{},
						Entries:        coinbaseEntry,
					},
				},
			},
			inputView: NewUtxoViewpoint(),
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.CoinbaseUTXOType, 0, false),
				},
			},
			err: false,
		},
		{
			// apply gas only tx, non-btm asset spent input will not be spent
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V1: 0},
							bc.Hash{V1: 1},
						},
						Entries: gasOnlyTxEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V1: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
					bc.Hash{V1: 1}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
				},
			},
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V1: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, true),
					bc.Hash{V1: 1}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
				},
			},
			gasOnlyTx: true,
			err:       false,
		},
		{
			// apply gas only tx, non-btm asset spent output will not be store
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{
								&bc.Hash{V1: 0},
								&bc.Hash{V1: 1},
							},
						},
						SpentOutputIDs: []bc.Hash{},
						Entries:        gasOnlyTxEntry,
					},
				},
			},
			inputView: NewUtxoViewpoint(),
			fetchView: NewUtxoViewpoint(),
			gasOnlyTx: true,
			err:       false,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: voteEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.VoteUTXOType, 0, false),
				},
			},
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.VoteUTXOType, 0, true),
				},
			},
			err: true,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					Height:            consensus.MainNetParams.VotePendingBlockNumber + 1,
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: voteEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.VoteUTXOType, 1, false),
				},
			},
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.VoteUTXOType, 1, true),
				},
			},
			err: false,
		},
	}

	for i, c := range cases {
		c.block.TransactionStatus.SetStatus(0, c.gasOnlyTx)
		if err := c.inputView.ApplyBlock(c.block, c.block.TransactionStatus); c.err != (err != nil) {
			t.Errorf("case #%d want err = %v, get err = %v", i, c.err, err)
		}
		if c.err {
			continue
		}
		if !testutil.DeepEqual(c.inputView, c.fetchView) {
			t.Errorf("test case %d, want %v, get %v", i, c.fetchView, c.inputView)
		}
	}
}

func TestDetachBlock(t *testing.T) {
	cases := []struct {
		block     *bc.Block
		inputView *UtxoViewpoint
		fetchView *UtxoViewpoint
		gasOnlyTx bool
		err       bool
	}{
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: NewUtxoViewpoint(),
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
				},
			},
			err: false,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{
								&bc.Hash{V0: 0},
								&bc.Hash{V0: 1},
							},
						},
						SpentOutputIDs: []bc.Hash{},
						Entries:        coinbaseEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
				},
			},
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, true),
				},
			},
			err: false,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
				},
			},
			err: true,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: defaultEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, true),
				},
			},
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
				},
			},
			err: false,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V1: 0},
							bc.Hash{V1: 1},
						},
						Entries: gasOnlyTxEntry,
					},
				},
			},
			inputView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V1: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, true),
					bc.Hash{V1: 1}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, true),
				},
			},
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V1: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
					bc.Hash{V1: 1}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, true),
				},
			},
			gasOnlyTx: true,
			err:       false,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{
								&bc.Hash{V1: 0},
								&bc.Hash{V1: 1},
							},
						},
						SpentOutputIDs: []bc.Hash{},
						Entries:        gasOnlyTxEntry,
					},
				},
			},
			inputView: NewUtxoViewpoint(),
			fetchView: NewUtxoViewpoint(),
			gasOnlyTx: true,
			err:       false,
		},
		{
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					TransactionStatus: bc.NewTransactionStatus(),
				},
				Transactions: []*bc.Tx{
					&bc.Tx{
						TxHeader: &bc.TxHeader{
							ResultIds: []*bc.Hash{},
						},
						SpentOutputIDs: []bc.Hash{
							bc.Hash{V0: 0},
						},
						Entries: voteEntry,
					},
				},
			},
			inputView: NewUtxoViewpoint(),
			fetchView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.VoteUTXOType, 0, false),
				},
			},
			err: false,
		},
	}

	for i, c := range cases {
		c.block.TransactionStatus.SetStatus(0, c.gasOnlyTx)
		if err := c.inputView.DetachBlock(c.block, c.block.TransactionStatus); c.err != (err != nil) {
			t.Errorf("case %d want err = %v, get err = %v", i, c.err, err)
		}
		if c.err {
			continue
		}
		if !testutil.DeepEqual(c.inputView, c.fetchView) {
			t.Errorf("test case %d, want %v, get %v", i, c.fetchView, c.inputView)
		}
	}
}

func TestApplyCrossChainUTXO(t *testing.T) {
	cases := []struct {
		desc         string
		block        *bc.Block
		tx           *bc.Tx
		prevUTXOView *UtxoViewpoint
		postUTXOView *UtxoViewpoint
		err          error
	}{
		{
			desc: "normal test",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					Height: 100,
				},
			},
			tx: &bc.Tx{
				TxHeader: &bc.TxHeader{
					ResultIds: []*bc.Hash{},
				},
				MainchainOutputIDs: []bc.Hash{
					bc.Hash{V0: 0},
				},
				Entries: voteEntry,
			},
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.CrosschainUTXOType, 0, false),
				},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.CrosschainUTXOType, 100, true),
				},
			},
			err: nil,
		},
		{
			desc: "test failed to find mainchain output entry",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{},
			},
			tx: &bc.Tx{
				TxHeader: &bc.TxHeader{
					ResultIds: []*bc.Hash{},
				},
				MainchainOutputIDs: []bc.Hash{
					bc.Hash{V0: 0},
				},
				Entries: voteEntry,
			},
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			err: errors.New("fail to find mainchain output entry"),
		},
		{
			desc: "test mainchain output has been spent",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{},
			},
			tx: &bc.Tx{
				TxHeader: &bc.TxHeader{
					ResultIds: []*bc.Hash{},
				},
				MainchainOutputIDs: []bc.Hash{
					bc.Hash{V0: 0},
				},
				Entries: voteEntry,
			},
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.CrosschainUTXOType, 0, true),
				},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			err: errors.New("mainchain output has been spent"),
		},
	}

	for i, c := range cases {
		if err := c.prevUTXOView.applyCrossChainUtxo(c.block, c.tx); err != nil {
			if err.Error() != c.err.Error() {
				t.Errorf("test case #%d want err = %v, got err = %v", i, c.err, err)
			}
			continue
		}

		if !testutil.DeepEqual(c.prevUTXOView, c.postUTXOView) {
			t.Errorf("test case #%d, want %v, got %v", i, c.postUTXOView, c.prevUTXOView)
		}
	}
}

func TestApplyOutputUTXO(t *testing.T) {
	cases := []struct {
		desc         string
		block        *bc.Block
		tx           *bc.Tx
		statusFail   bool
		prevUTXOView *UtxoViewpoint
		postUTXOView *UtxoViewpoint
		err          error
	}{
		{
			desc: "normal test IntraChainOutput,VoteOutput,Retirement",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{},
			},
			tx: &bc.Tx{
				TxHeader: &bc.TxHeader{
					ResultIds: []*bc.Hash{&bc.Hash{V0: 0}, &bc.Hash{V0: 1}, &bc.Hash{V0: 2}},
				},
				Entries: map[bc.Hash]bc.Entry{
					bc.Hash{V0: 0}: &bc.IntraChainOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 0},
								Amount:  100,
							},
						},
					},
					bc.Hash{V0: 1}: &bc.VoteOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 1},
							},
						},
					},
					bc.Hash{V0: 2}: &bc.Retirement{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 1},
							},
						},
					},
				},
			},
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.VoteUTXOType, 0, false),
				},
			},
			err: nil,
		},
		{
			desc: "test statusFail",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{},
			},
			tx: &bc.Tx{
				TxHeader: &bc.TxHeader{
					ResultIds: []*bc.Hash{&bc.Hash{V0: 0}, &bc.Hash{V0: 1}, &bc.Hash{V0: 2}},
				},
				Entries: map[bc.Hash]bc.Entry{
					bc.Hash{V0: 0}: &bc.IntraChainOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 0},
								Amount:  100,
							},
						},
					},
					bc.Hash{V0: 1}: &bc.VoteOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: consensus.BTMAssetID,
							},
						},
					},
					bc.Hash{V0: 2}: &bc.Retirement{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 1},
							},
						},
					},
				},
			},
			statusFail: true,
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.VoteUTXOType, 0, false),
				},
			},
			err: nil,
		},
		{
			desc: "test failed on found id from tx entry",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{},
			},
			tx: &bc.Tx{
				TxHeader: &bc.TxHeader{
					ResultIds: []*bc.Hash{&bc.Hash{V0: 0}, &bc.Hash{V0: 1}, &bc.Hash{V0: 2}},
				},
				Entries: map[bc.Hash]bc.Entry{
					bc.Hash{V0: 1}: &bc.VoteOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: consensus.BTMAssetID,
							},
						},
					},
					bc.Hash{V0: 2}: &bc.Retirement{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 1},
							},
						},
					},
				},
			},
			statusFail: false,
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			err: bc.ErrMissingEntry,
		},
	}

	for i, c := range cases {
		if err := c.prevUTXOView.applyOutputUtxo(c.block, c.tx, c.statusFail); err != nil {
			if errors.Root(err) != errors.Root(c.err) {
				t.Errorf("test case #%d want err = %v, got err = %v", i, c.err.Error(), err.Error())
			}
			continue
		}

		if !testutil.DeepEqual(c.prevUTXOView, c.postUTXOView) {
			t.Errorf("test case #%d, want %v, got %v", i, c.postUTXOView, c.prevUTXOView)
		}
	}
}

func TestApplySpendUTXO(t *testing.T) {
	cases := []struct {
		desc         string
		block        *bc.Block
		tx           *bc.Tx
		statusFail   bool
		prevUTXOView *UtxoViewpoint
		postUTXOView *UtxoViewpoint
		err          error
	}{
		{
			desc: "normal test",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					Height: consensus.ActiveNetParams.VotePendingBlockNumber,
				},
			},
			tx: &bc.Tx{
				TxHeader:       &bc.TxHeader{},
				SpentOutputIDs: []bc.Hash{{V0: 0}, {V0: 1}, {V0: 2}},
				Entries: map[bc.Hash]bc.Entry{
					bc.Hash{V0: 0}: &bc.IntraChainOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 0},
								Amount:  100,
							},
						},
					},
					bc.Hash{V0: 1}: &bc.VoteOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 1},
							},
						},
					},
					bc.Hash{V0: 2}: &bc.IntraChainOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: consensus.BTMAssetID,
								Amount:  100,
							},
						},
					},
				},
			},
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.VoteUTXOType, 0, false),
					bc.Hash{V0: 2}: storage.NewUtxoEntry(storage.CoinbaseUTXOType, 0, false),
				},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, true),
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.VoteUTXOType, 0, true),
					bc.Hash{V0: 2}: storage.NewUtxoEntry(storage.CoinbaseUTXOType, 0, true),
				},
			},
			err: nil,
		},
		{
			desc: "test coinbase is not ready for use",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					Height: consensus.ActiveNetParams.CoinbasePendingBlockNumber - 1,
				},
			},
			tx: &bc.Tx{
				TxHeader:       &bc.TxHeader{},
				SpentOutputIDs: []bc.Hash{{V0: 1}, {V0: 2}},
				Entries: map[bc.Hash]bc.Entry{
					bc.Hash{V0: 1}: &bc.VoteOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 1},
							},
						},
					},
					bc.Hash{V0: 2}: &bc.IntraChainOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: consensus.BTMAssetID,
								Amount:  100,
							},
						},
					},
				},
			},
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
					bc.Hash{V0: 2}: storage.NewUtxoEntry(storage.CoinbaseUTXOType, 0, false),
				},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			err: errors.New("coinbase utxo is not ready for use"),
		},
		{
			desc: "test Coin is  within the voting lock time",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					Height: consensus.ActiveNetParams.VotePendingBlockNumber - 1,
				},
			},
			tx: &bc.Tx{
				TxHeader:       &bc.TxHeader{},
				SpentOutputIDs: []bc.Hash{{V0: 1}, {V0: 2}},
				Entries: map[bc.Hash]bc.Entry{
					bc.Hash{V0: 1}: &bc.VoteOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 1},
							},
						},
					},
					bc.Hash{V0: 2}: &bc.IntraChainOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: consensus.BTMAssetID,
								Amount:  100,
							},
						},
					},
				},
			},
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
					bc.Hash{V0: 2}: storage.NewUtxoEntry(storage.VoteUTXOType, 0, false),
				},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			err: errors.New("Coin is  within the voting lock time"),
		},
		{
			desc: "test utxo has been spent",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					Height: 0,
				},
			},
			tx: &bc.Tx{
				TxHeader:       &bc.TxHeader{},
				SpentOutputIDs: []bc.Hash{{V0: 0}, {V0: 1}, {V0: 2}},
				Entries: map[bc.Hash]bc.Entry{
					bc.Hash{V0: 0}: &bc.IntraChainOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 0},
								Amount:  100,
							},
						},
					},
					bc.Hash{V0: 1}: &bc.VoteOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 1},
							},
						},
					},
					bc.Hash{V0: 2}: &bc.IntraChainOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: consensus.BTMAssetID,
								Amount:  100,
							},
						},
					},
				},
			},
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, true),
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.VoteUTXOType, 0, false),
					bc.Hash{V0: 2}: storage.NewUtxoEntry(storage.CoinbaseUTXOType, 0, false),
				},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			err: errors.New("utxo has been spent"),
		},
		{
			desc: "test faild to find utxo entry",
			block: &bc.Block{
				BlockHeader: &bc.BlockHeader{
					Height: 0,
				},
			},
			tx: &bc.Tx{
				TxHeader:       &bc.TxHeader{},
				SpentOutputIDs: []bc.Hash{{V0: 0}, {V0: 1}, {V0: 2}},
				Entries: map[bc.Hash]bc.Entry{
					bc.Hash{V0: 0}: &bc.IntraChainOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 0},
								Amount:  100,
							},
						},
					},
					bc.Hash{V0: 1}: &bc.VoteOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 1},
							},
						},
					},
					bc.Hash{V0: 2}: &bc.IntraChainOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: consensus.BTMAssetID,
								Amount:  100,
							},
						},
					},
				},
			},
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.VoteUTXOType, 0, false),
					bc.Hash{V0: 2}: storage.NewUtxoEntry(storage.CoinbaseUTXOType, 0, false),
				},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			err: errors.New("fail to find utxo entry"),
		},
	}

	for i, c := range cases {
		if err := c.prevUTXOView.applySpendUtxo(c.block, c.tx, c.statusFail); err != nil {
			if err.Error() != c.err.Error() {
				t.Errorf("test case #%d want err = %v, got err = %v", i, err.Error(), c.err.Error())
			}
			continue
		}

		if !testutil.DeepEqual(c.prevUTXOView, c.postUTXOView) {
			t.Errorf("test case #%d, want %v, got %v", i, spew.Sdump(c.postUTXOView), spew.Sdump(c.prevUTXOView))
		}
	}
}

func TestDetachCrossChainUTXO(t *testing.T) {
	cases := []struct {
		desc         string
		tx           *bc.Tx
		prevUTXOView *UtxoViewpoint
		postUTXOView *UtxoViewpoint
		err          error
	}{
		{
			desc: "normal test",
			tx: &bc.Tx{
				TxHeader: &bc.TxHeader{
					ResultIds: []*bc.Hash{},
				},
				MainchainOutputIDs: []bc.Hash{
					bc.Hash{V0: 0},
				},
				Entries: voteEntry,
			},
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.CrosschainUTXOType, 0, true),
				},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.CrosschainUTXOType, 0, false),
				},
			},
			err: nil,
		},
		{
			desc: "test failed to find mainchain output entry",
			tx: &bc.Tx{
				TxHeader: &bc.TxHeader{
					ResultIds: []*bc.Hash{},
				},
				MainchainOutputIDs: []bc.Hash{
					bc.Hash{V0: 0},
				},
				Entries: voteEntry,
			},
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			err: errors.New("fail to find mainchain output entry"),
		},
		{
			desc: "test revert output is unspent",
			tx: &bc.Tx{
				TxHeader: &bc.TxHeader{
					ResultIds: []*bc.Hash{},
				},
				MainchainOutputIDs: []bc.Hash{
					bc.Hash{V0: 0},
				},
				Entries: voteEntry,
			},
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.CrosschainUTXOType, 0, false),
				},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			err: errors.New("mainchain output is unspent"),
		},
	}

	for i, c := range cases {
		if err := c.prevUTXOView.detachCrossChainUtxo(c.tx); err != nil {
			if err.Error() != c.err.Error() {
				t.Errorf("test case #%d want err = %v, got err = %v", i, c.err, err)
			}
			continue
		}

		if !testutil.DeepEqual(c.prevUTXOView, c.postUTXOView) {
			t.Errorf("test case #%d, want %v, got %v", i, c.postUTXOView, c.prevUTXOView)
		}
	}
}

func TestDetachOutputUTXO(t *testing.T) {
	cases := []struct {
		desc         string
		tx           *bc.Tx
		statusFail   bool
		prevUTXOView *UtxoViewpoint
		postUTXOView *UtxoViewpoint
		err          error
	}{
		{
			desc: "normal test IntraChainOutput,VoteOutput",
			tx: &bc.Tx{
				TxHeader: &bc.TxHeader{
					ResultIds: []*bc.Hash{&bc.Hash{V0: 0}, &bc.Hash{V0: 1}, &bc.Hash{V0: 2}},
				},
				Entries: map[bc.Hash]bc.Entry{
					bc.Hash{V0: 0}: &bc.IntraChainOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 0},
								Amount:  100,
							},
						},
					},
					bc.Hash{V0: 1}: &bc.VoteOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 1},
							},
						},
					},
					bc.Hash{V0: 2}: &bc.Retirement{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 1},
							},
						},
					},
				},
			},
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, true),
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.VoteUTXOType, 0, true),
				},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, true),
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.VoteUTXOType, 0, true),
				},
			},
			err: nil,
		},
		{
			desc: "test statusFail",
			tx: &bc.Tx{
				TxHeader: &bc.TxHeader{
					ResultIds: []*bc.Hash{&bc.Hash{V0: 0}, &bc.Hash{V0: 1}},
				},
				Entries: map[bc.Hash]bc.Entry{
					bc.Hash{V0: 0}: &bc.IntraChainOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 0},
								Amount:  100,
							},
						},
					},
					bc.Hash{V0: 1}: &bc.VoteOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: consensus.BTMAssetID,
							},
						},
					},
				},
			},
			statusFail: true,
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.VoteUTXOType, 0, true),
				},
			},
			err: nil,
		},
		{
			desc: "test failed on found id from tx entry",
			tx: &bc.Tx{
				TxHeader: &bc.TxHeader{
					ResultIds: []*bc.Hash{&bc.Hash{V0: 0}, &bc.Hash{V0: 1}, &bc.Hash{V0: 2}},
				},
				Entries: map[bc.Hash]bc.Entry{
					bc.Hash{V0: 1}: &bc.VoteOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: consensus.BTMAssetID,
							},
						},
					},
					bc.Hash{V0: 2}: &bc.Retirement{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 1},
							},
						},
					},
				},
			},
			statusFail: false,
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			err: bc.ErrMissingEntry,
		},
	}

	for i, c := range cases {
		if err := c.prevUTXOView.detachOutputUtxo(c.tx, c.statusFail); err != nil {
			if errors.Root(err) != errors.Root(c.err) {
				t.Errorf("test case #%d want err = %v, got err = %v", i, c.err.Error(), err.Error())
			}
			continue
		}

		if !testutil.DeepEqual(c.prevUTXOView, c.postUTXOView) {
			t.Errorf("test case #%d, want %v, got %v", i, c.postUTXOView, c.prevUTXOView)
		}
	}
}

func TestDetachSpendUTXO(t *testing.T) {
	cases := []struct {
		desc         string
		tx           *bc.Tx
		statusFail   bool
		prevUTXOView *UtxoViewpoint
		postUTXOView *UtxoViewpoint
		err          error
	}{
		{
			desc: "normal test",
			tx: &bc.Tx{
				TxHeader:       &bc.TxHeader{},
				SpentOutputIDs: []bc.Hash{{V0: 0}, {V0: 1}},
				Entries: map[bc.Hash]bc.Entry{
					bc.Hash{V0: 0}: &bc.IntraChainOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 0},
								Amount:  100,
							},
						},
					},
					bc.Hash{V0: 1}: &bc.VoteOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 1},
							},
						},
					},
				},
			},
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, true),
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.VoteUTXOType, 0, true),
				},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.VoteUTXOType, 0, false),
				},
			},
			err: nil,
		},
		{
			desc: "test utxo has been spent",
			tx: &bc.Tx{
				TxHeader:       &bc.TxHeader{},
				SpentOutputIDs: []bc.Hash{{V0: 0}, {V0: 1}, {V0: 2}},
				Entries: map[bc.Hash]bc.Entry{
					bc.Hash{V0: 0}: &bc.IntraChainOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: consensus.BTMAssetID,
								Amount:  100,
							},
						},
					},
					bc.Hash{V0: 1}: &bc.VoteOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: &bc.AssetID{V0: 1},
							},
						},
					},
					bc.Hash{V0: 2}: &bc.IntraChainOutput{
						Source: &bc.ValueSource{
							Value: &bc.AssetAmount{
								AssetId: consensus.BTMAssetID,
								Amount:  100,
							},
						},
					},
				},
			},
			prevUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{
					bc.Hash{V0: 0}: storage.NewUtxoEntry(storage.NormalUTXOType, 0, false),
					bc.Hash{V0: 1}: storage.NewUtxoEntry(storage.VoteUTXOType, 0, true),
				},
			},
			postUTXOView: &UtxoViewpoint{
				Entries: map[bc.Hash]*storage.UtxoEntry{},
			},
			err: errors.New("try to revert an unspent utxo"),
		},
	}

	for i, c := range cases {
		if err := c.prevUTXOView.detachSpendUtxo(c.tx, c.statusFail); err != nil {
			if err.Error() != c.err.Error() {
				t.Errorf("test case #%d want err = %v, got err = %v", i, err.Error(), c.err.Error())
			}
			continue
		}

		if !testutil.DeepEqual(c.prevUTXOView, c.postUTXOView) {
			t.Errorf("test case #%d, want %v, got %v", i, spew.Sdump(c.postUTXOView), spew.Sdump(c.prevUTXOView))
		}
	}
}
