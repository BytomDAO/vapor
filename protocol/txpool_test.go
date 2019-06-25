package protocol

import (
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/vapor/consensus"
	"github.com/vapor/database/storage"
	"github.com/vapor/event"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
	"github.com/vapor/testutil"
)

var testTxs = []*types.Tx{
	//tx0
	types.NewTx(types.TxData{
		SerializedSize: 100,
		Inputs: []*types.TxInput{
			types.NewSpendInput(nil, bc.NewHash([32]byte{0x01}), *consensus.BTMAssetID, 1, 1, []byte{0x51}),
		},
		Outputs: []*types.TxOutput{
			types.NewIntraChainOutput(*consensus.BTMAssetID, 1, []byte{0x6a}),
		},
	}),
	//tx1
	types.NewTx(types.TxData{
		SerializedSize: 100,
		Inputs: []*types.TxInput{
			types.NewSpendInput(nil, bc.NewHash([32]byte{0x01}), *consensus.BTMAssetID, 1, 1, []byte{0x51}),
		},
		Outputs: []*types.TxOutput{
			types.NewIntraChainOutput(*consensus.BTMAssetID, 1, []byte{0x6b}),
		},
	}),
	//tx2
	types.NewTx(types.TxData{
		SerializedSize: 150,
		TimeRange:      0,
		Inputs: []*types.TxInput{
			types.NewSpendInput(nil, bc.NewHash([32]byte{0x01}), *consensus.BTMAssetID, 1, 1, []byte{0x51}),
			types.NewSpendInput(nil, bc.NewHash([32]byte{0x02}), bc.NewAssetID([32]byte{0xa1}), 4, 1, []byte{0x51}),
		},
		Outputs: []*types.TxOutput{
			types.NewIntraChainOutput(*consensus.BTMAssetID, 1, []byte{0x6b}),
			types.NewIntraChainOutput(bc.NewAssetID([32]byte{0xa1}), 4, []byte{0x61}),
		},
	}),
	//tx3
	types.NewTx(types.TxData{

		SerializedSize: 100,
		Inputs: []*types.TxInput{
			types.NewSpendInput(nil, testutil.MustDecodeHash("7d3f8e8474775f9fab2a7370529f0569a2199b22a5a83d235a036f50de3e8c84"), bc.NewAssetID([32]byte{0xa1}), 4, 1, []byte{0x61}),
		},
		Outputs: []*types.TxOutput{
			types.NewIntraChainOutput(bc.NewAssetID([32]byte{0xa1}), 3, []byte{0x62}),
			types.NewIntraChainOutput(bc.NewAssetID([32]byte{0xa1}), 1, []byte{0x63}),
		},
	}),
	//tx4
	types.NewTx(types.TxData{
		SerializedSize: 100,
		Inputs: []*types.TxInput{
			types.NewSpendInput(nil, testutil.MustDecodeHash("9a26cde504a5d7190dbed119280276f9816d9c2b7d20c768b312be57930fe840"), bc.NewAssetID([32]byte{0xa1}), 3, 0, []byte{0x62}),
		},
		Outputs: []*types.TxOutput{
			types.NewIntraChainOutput(bc.NewAssetID([32]byte{0xa1}), 2, []byte{0x64}),
			types.NewIntraChainOutput(bc.NewAssetID([32]byte{0xa1}), 1, []byte{0x65}),
		},
	}),
	//tx5
	types.NewTx(types.TxData{
		SerializedSize: 100,
		Inputs: []*types.TxInput{
			types.NewSpendInput(nil, bc.NewHash([32]byte{0x01}), *consensus.BTMAssetID, 1, 1, []byte{0x51}),
		},
		Outputs: []*types.TxOutput{
			types.NewIntraChainOutput(*consensus.BTMAssetID, 0, []byte{0x51}),
		},
	}),
	//tx6
	types.NewTx(types.TxData{
		SerializedSize: 100,
		Inputs: []*types.TxInput{
			types.NewSpendInput(nil, bc.NewHash([32]byte{0x01}), *consensus.BTMAssetID, 3, 1, []byte{0x51}),
			types.NewSpendInput(nil, testutil.MustDecodeHash("9a26cde504a5d7190dbed119280276f9816d9c2b7d20c768b312be57930fe840"), bc.NewAssetID([32]byte{0xa1}), 3, 0, []byte{0x62}),
		},
		Outputs: []*types.TxOutput{
			types.NewIntraChainOutput(*consensus.BTMAssetID, 2, []byte{0x51}),
			types.NewIntraChainOutput(bc.NewAssetID([32]byte{0xa1}), 0, []byte{0x65}),
		},
	}),
	//tx7
	types.NewTx(types.TxData{
		SerializedSize: 150,
		TimeRange:      0,
		Inputs: []*types.TxInput{
			types.NewSpendInput(nil, bc.NewHash([32]byte{0x01}), *consensus.BTMAssetID, 1, 1, []byte{0x51}),
			types.NewSpendInput(nil, bc.NewHash([32]byte{0x02}), bc.NewAssetID([32]byte{0xa1}), 4, 1, []byte{0x51}),
		},
		Outputs: []*types.TxOutput{
			types.NewIntraChainOutput(*consensus.BTMAssetID, 1, []byte{0x6b}),
			types.NewVoteOutput(bc.NewAssetID([32]byte{0xa1}), 4, []byte{0x61}, []byte("a8f410b9f7cd9ce352d215ed17c85559c351dc8d18ed89ad403ca28cfc423f612e04a1c9584f945c286c47ec1e5b8405c65ff56e31f44a2627aca4f77e03936f")),
		},
	}),
}

type mockStore struct{}

func (s *mockStore) BlockExist(hash *bc.Hash) bool                                { return false }
func (s *mockStore) GetBlock(*bc.Hash) (*types.Block, error)                      { return nil, nil }
func (s *mockStore) GetBlockHeader(*bc.Hash) (*types.BlockHeader, error)          { return nil, nil }
func (s *mockStore) GetStoreStatus() *BlockStoreState                             { return nil }
func (s *mockStore) GetTransactionStatus(*bc.Hash) (*bc.TransactionStatus, error) { return nil, nil }
func (s *mockStore) GetTransactionsUtxo(*state.UtxoViewpoint, []*bc.Tx) error     { return nil }
func (s *mockStore) GetUtxo(*bc.Hash) (*storage.UtxoEntry, error)                 { return nil, nil }
func (s *mockStore) GetVoteResult(uint64) (*state.VoteResult, error)              { return nil, nil }
func (s *mockStore) GetMainChainHash(uint64) (*bc.Hash, error)                    { return nil, nil }
func (s *mockStore) GetBlockHashesByHeight(uint64) ([]*bc.Hash, error)            { return nil, nil }
func (s *mockStore) SaveBlock(*types.Block, *bc.TransactionStatus) error          { return nil }
func (s *mockStore) SaveBlockHeader(*types.BlockHeader) error                     { return nil }
func (s *mockStore) SaveChainStatus(*types.BlockHeader, *types.BlockHeader, []*types.BlockHeader, *state.UtxoViewpoint, []*state.VoteResult) error {
	return nil
}

func TestAddOrphan(t *testing.T) {
	cases := []struct {
		before         *TxPool
		after          *TxPool
		addOrphan      *TxDesc
		requireParents []*bc.Hash
	}{
		{
			before: &TxPool{
				orphans:       map[bc.Hash]*orphanTx{},
				orphansByPrev: map[bc.Hash]map[bc.Hash]*orphanTx{},
			},
			after: &TxPool{
				orphans: map[bc.Hash]*orphanTx{
					testTxs[0].ID: {
						TxDesc: &TxDesc{
							Tx: testTxs[0],
						},
					},
				},
				orphansByPrev: map[bc.Hash]map[bc.Hash]*orphanTx{
					testTxs[0].SpentOutputIDs[0]: {
						testTxs[0].ID: {
							TxDesc: &TxDesc{
								Tx: testTxs[0],
							},
						},
					},
				},
			},
			addOrphan:      &TxDesc{Tx: testTxs[0]},
			requireParents: []*bc.Hash{&testTxs[0].SpentOutputIDs[0]},
		},
		{
			before: &TxPool{
				orphans: map[bc.Hash]*orphanTx{
					testTxs[0].ID: {
						TxDesc: &TxDesc{
							Tx: testTxs[0],
						},
					},
				},
				orphansByPrev: map[bc.Hash]map[bc.Hash]*orphanTx{
					testTxs[0].SpentOutputIDs[0]: {
						testTxs[0].ID: {
							TxDesc: &TxDesc{
								Tx: testTxs[0],
							},
						},
					},
				},
			},
			after: &TxPool{
				orphans: map[bc.Hash]*orphanTx{
					testTxs[0].ID: {
						TxDesc: &TxDesc{
							Tx: testTxs[0],
						},
					},
					testTxs[1].ID: {
						TxDesc: &TxDesc{
							Tx: testTxs[1],
						},
					},
				},
				orphansByPrev: map[bc.Hash]map[bc.Hash]*orphanTx{
					testTxs[0].SpentOutputIDs[0]: {
						testTxs[0].ID: {
							TxDesc: &TxDesc{
								Tx: testTxs[0],
							},
						},
						testTxs[1].ID: {
							TxDesc: &TxDesc{
								Tx: testTxs[1],
							},
						},
					},
				},
			},
			addOrphan:      &TxDesc{Tx: testTxs[1]},
			requireParents: []*bc.Hash{&testTxs[1].SpentOutputIDs[0]},
		},
		{
			before: &TxPool{
				orphans:       map[bc.Hash]*orphanTx{},
				orphansByPrev: map[bc.Hash]map[bc.Hash]*orphanTx{},
			},
			after: &TxPool{
				orphans: map[bc.Hash]*orphanTx{
					testTxs[2].ID: {
						TxDesc: &TxDesc{
							Tx: testTxs[2],
						},
					},
				},
				orphansByPrev: map[bc.Hash]map[bc.Hash]*orphanTx{
					testTxs[2].SpentOutputIDs[1]: {
						testTxs[2].ID: {
							TxDesc: &TxDesc{
								Tx: testTxs[2],
							},
						},
					},
				},
			},
			addOrphan:      &TxDesc{Tx: testTxs[2]},
			requireParents: []*bc.Hash{&testTxs[2].SpentOutputIDs[1]},
		},
	}

	for i, c := range cases {
		c.before.addOrphan(c.addOrphan, c.requireParents)
		for _, orphan := range c.before.orphans {
			orphan.expiration = time.Time{}
		}
		for _, orphans := range c.before.orphansByPrev {
			for _, orphan := range orphans {
				orphan.expiration = time.Time{}
			}
		}
		if !testutil.DeepEqual(c.before, c.after) {
			t.Errorf("case %d: got %v want %v", i, c.before, c.after)
		}
	}
}

func TestAddTransaction(t *testing.T) {
	dispatcher := event.NewDispatcher()
	cases := []struct {
		before *TxPool
		after  *TxPool
		addTx  *TxDesc
	}{
		{
			before: &TxPool{
				pool:            map[bc.Hash]*TxDesc{},
				utxo:            map[bc.Hash]*types.Tx{},
				eventDispatcher: dispatcher,
			},
			after: &TxPool{
				pool: map[bc.Hash]*TxDesc{
					testTxs[2].ID: {
						Tx:         testTxs[2],
						StatusFail: false,
					},
				},
				utxo: map[bc.Hash]*types.Tx{
					*testTxs[2].ResultIds[0]: testTxs[2],
					*testTxs[2].ResultIds[1]: testTxs[2],
				},
			},
			addTx: &TxDesc{
				Tx:         testTxs[2],
				StatusFail: false,
			},
		},
		{
			before: &TxPool{
				pool:            map[bc.Hash]*TxDesc{},
				utxo:            map[bc.Hash]*types.Tx{},
				eventDispatcher: dispatcher,
			},
			after: &TxPool{
				pool: map[bc.Hash]*TxDesc{
					testTxs[2].ID: {
						Tx:         testTxs[2],
						StatusFail: true,
					},
				},
				utxo: map[bc.Hash]*types.Tx{
					*testTxs[2].ResultIds[0]: testTxs[2],
				},
			},
			addTx: &TxDesc{
				Tx:         testTxs[2],
				StatusFail: true,
			},
		},
		{
			before: &TxPool{
				pool:            map[bc.Hash]*TxDesc{},
				utxo:            map[bc.Hash]*types.Tx{},
				eventDispatcher: dispatcher,
			},
			after: &TxPool{
				pool: map[bc.Hash]*TxDesc{
					testTxs[7].ID: {
						Tx:         testTxs[7],
						StatusFail: false,
					},
				},
				utxo: map[bc.Hash]*types.Tx{
					*testTxs[7].ResultIds[0]: testTxs[7],
					*testTxs[7].ResultIds[1]: testTxs[7],
				},
			},
			addTx: &TxDesc{
				Tx:         testTxs[7],
				StatusFail: false,
			},
		},
		{
			before: &TxPool{
				pool:            map[bc.Hash]*TxDesc{},
				utxo:            map[bc.Hash]*types.Tx{},
				eventDispatcher: dispatcher,
			},
			after: &TxPool{
				pool: map[bc.Hash]*TxDesc{
					testTxs[7].ID: {
						Tx:         testTxs[7],
						StatusFail: true,
					},
				},
				utxo: map[bc.Hash]*types.Tx{
					*testTxs[7].ResultIds[0]: testTxs[7],
				},
			},
			addTx: &TxDesc{
				Tx:         testTxs[7],
				StatusFail: true,
			},
		},
	}

	for i, c := range cases {
		c.before.addTransaction(c.addTx)
		for _, txD := range c.before.pool {
			txD.Added = time.Time{}
		}
		if !testutil.DeepEqual(c.before.pool, c.after.pool) {
			t.Errorf("case %d: got %v want %v", i, c.before.pool, c.after.pool)
		}
		if !testutil.DeepEqual(c.before.utxo, c.after.utxo) {
			t.Errorf("case %d: got %v want %v", i, c.before.utxo, c.after.utxo)
		}
	}
}

func TestExpireOrphan(t *testing.T) {
	before := &TxPool{
		orphans: map[bc.Hash]*orphanTx{
			testTxs[0].ID: {
				expiration: time.Unix(1533489701, 0),
				TxDesc: &TxDesc{
					Tx: testTxs[0],
				},
			},
			testTxs[1].ID: {
				expiration: time.Unix(1633489701, 0),
				TxDesc: &TxDesc{
					Tx: testTxs[1],
				},
			},
		},
		orphansByPrev: map[bc.Hash]map[bc.Hash]*orphanTx{
			testTxs[0].SpentOutputIDs[0]: {
				testTxs[0].ID: {
					expiration: time.Unix(1533489701, 0),
					TxDesc: &TxDesc{
						Tx: testTxs[0],
					},
				},
				testTxs[1].ID: {
					expiration: time.Unix(1633489701, 0),
					TxDesc: &TxDesc{
						Tx: testTxs[1],
					},
				},
			},
		},
	}

	want := &TxPool{
		orphans: map[bc.Hash]*orphanTx{
			testTxs[1].ID: {
				expiration: time.Unix(1633489701, 0),
				TxDesc: &TxDesc{
					Tx: testTxs[1],
				},
			},
		},
		orphansByPrev: map[bc.Hash]map[bc.Hash]*orphanTx{
			testTxs[0].SpentOutputIDs[0]: {
				testTxs[1].ID: {
					expiration: time.Unix(1633489701, 0),
					TxDesc: &TxDesc{
						Tx: testTxs[1],
					},
				},
			},
		},
	}

	before.ExpireOrphan(time.Unix(1633479701, 0))
	if !testutil.DeepEqual(before, want) {
		t.Errorf("got %v want %v", before, want)
	}
}

func TestProcessOrphans(t *testing.T) {
	dispatcher := event.NewDispatcher()
	cases := []struct {
		before    *TxPool
		after     *TxPool
		processTx *TxDesc
	}{
		{
			before: &TxPool{
				pool:            map[bc.Hash]*TxDesc{},
				utxo:            map[bc.Hash]*types.Tx{},
				eventDispatcher: dispatcher,
				orphans: map[bc.Hash]*orphanTx{
					testTxs[3].ID: {
						TxDesc: &TxDesc{
							Tx: testTxs[3],
						},
					},
				},
				orphansByPrev: map[bc.Hash]map[bc.Hash]*orphanTx{
					testTxs[3].SpentOutputIDs[0]: {
						testTxs[3].ID: {
							TxDesc: &TxDesc{
								Tx: testTxs[3],
							},
						},
					},
				},
			},
			after: &TxPool{
				pool: map[bc.Hash]*TxDesc{
					testTxs[3].ID: {
						Tx:         testTxs[3],
						StatusFail: false,
					},
				},
				utxo: map[bc.Hash]*types.Tx{
					*testTxs[3].ResultIds[0]: testTxs[3],
					*testTxs[3].ResultIds[1]: testTxs[3],
				},
				eventDispatcher: dispatcher,
				orphans:         map[bc.Hash]*orphanTx{},
				orphansByPrev:   map[bc.Hash]map[bc.Hash]*orphanTx{},
			},
			processTx: &TxDesc{Tx: testTxs[2]},
		},
		{
			before: &TxPool{
				pool:            map[bc.Hash]*TxDesc{},
				utxo:            map[bc.Hash]*types.Tx{},
				eventDispatcher: dispatcher,
				orphans: map[bc.Hash]*orphanTx{
					testTxs[3].ID: {
						TxDesc: &TxDesc{
							Tx: testTxs[3],
						},
					},
					testTxs[4].ID: {
						TxDesc: &TxDesc{
							Tx: testTxs[4],
						},
					},
				},
				orphansByPrev: map[bc.Hash]map[bc.Hash]*orphanTx{
					testTxs[3].SpentOutputIDs[0]: {
						testTxs[3].ID: {
							TxDesc: &TxDesc{
								Tx: testTxs[3],
							},
						},
					},
					testTxs[4].SpentOutputIDs[0]: {
						testTxs[4].ID: {
							TxDesc: &TxDesc{
								Tx: testTxs[4],
							},
						},
					},
				},
			},
			after: &TxPool{
				pool: map[bc.Hash]*TxDesc{
					testTxs[3].ID: {
						Tx:         testTxs[3],
						StatusFail: false,
					},
					testTxs[4].ID: {
						Tx:         testTxs[4],
						StatusFail: false,
					},
				},
				utxo: map[bc.Hash]*types.Tx{
					*testTxs[3].ResultIds[0]: testTxs[3],
					*testTxs[3].ResultIds[1]: testTxs[3],
					*testTxs[4].ResultIds[0]: testTxs[4],
					*testTxs[4].ResultIds[1]: testTxs[4],
				},
				eventDispatcher: dispatcher,
				orphans:         map[bc.Hash]*orphanTx{},
				orphansByPrev:   map[bc.Hash]map[bc.Hash]*orphanTx{},
			},
			processTx: &TxDesc{Tx: testTxs[2]},
		},
	}

	for i, c := range cases {
		c.before.store = &mockStore{}
		c.before.addTransaction(c.processTx)
		c.before.processOrphans(c.processTx)
		c.before.RemoveTransaction(&c.processTx.Tx.ID)
		c.before.store = nil
		c.before.lastUpdated = 0
		for _, txD := range c.before.pool {
			txD.Added = time.Time{}
		}

		if !testutil.DeepEqual(c.before, c.after) {
			t.Errorf("case %d: got %v want %v", i, c.before, c.after)
		}
	}
}

func TestRemoveOrphan(t *testing.T) {
	cases := []struct {
		before       *TxPool
		after        *TxPool
		removeHashes []*bc.Hash
	}{
		{
			before: &TxPool{
				orphans: map[bc.Hash]*orphanTx{
					testTxs[0].ID: {
						expiration: time.Unix(1533489701, 0),
						TxDesc: &TxDesc{
							Tx: testTxs[0],
						},
					},
				},
				orphansByPrev: map[bc.Hash]map[bc.Hash]*orphanTx{
					testTxs[0].SpentOutputIDs[0]: {
						testTxs[0].ID: {
							expiration: time.Unix(1533489701, 0),
							TxDesc: &TxDesc{
								Tx: testTxs[0],
							},
						},
					},
				},
			},
			after: &TxPool{
				orphans:       map[bc.Hash]*orphanTx{},
				orphansByPrev: map[bc.Hash]map[bc.Hash]*orphanTx{},
			},
			removeHashes: []*bc.Hash{
				&testTxs[0].ID,
			},
		},
		{
			before: &TxPool{
				orphans: map[bc.Hash]*orphanTx{
					testTxs[0].ID: {
						expiration: time.Unix(1533489701, 0),
						TxDesc: &TxDesc{
							Tx: testTxs[0],
						},
					},
					testTxs[1].ID: {
						expiration: time.Unix(1533489701, 0),
						TxDesc: &TxDesc{
							Tx: testTxs[1],
						},
					},
				},
				orphansByPrev: map[bc.Hash]map[bc.Hash]*orphanTx{
					testTxs[0].SpentOutputIDs[0]: {
						testTxs[0].ID: {
							expiration: time.Unix(1533489701, 0),
							TxDesc: &TxDesc{
								Tx: testTxs[0],
							},
						},
						testTxs[1].ID: {
							expiration: time.Unix(1533489701, 0),
							TxDesc: &TxDesc{
								Tx: testTxs[1],
							},
						},
					},
				},
			},
			after: &TxPool{
				orphans: map[bc.Hash]*orphanTx{
					testTxs[0].ID: {
						expiration: time.Unix(1533489701, 0),
						TxDesc: &TxDesc{
							Tx: testTxs[0],
						},
					},
				},
				orphansByPrev: map[bc.Hash]map[bc.Hash]*orphanTx{
					testTxs[0].SpentOutputIDs[0]: {
						testTxs[0].ID: {
							expiration: time.Unix(1533489701, 0),
							TxDesc: &TxDesc{
								Tx: testTxs[0],
							},
						},
					},
				},
			},
			removeHashes: []*bc.Hash{
				&testTxs[1].ID,
			},
		},
	}

	for i, c := range cases {
		for _, hash := range c.removeHashes {
			c.before.removeOrphan(hash)
		}
		if !testutil.DeepEqual(c.before, c.after) {
			t.Errorf("case %d: got %v want %v", i, c.before, c.after)
		}
	}
}

type mockStore1 struct{}

func (s *mockStore1) BlockExist(hash *bc.Hash) bool                                { return false }
func (s *mockStore1) GetBlock(*bc.Hash) (*types.Block, error)                      { return nil, nil }
func (s *mockStore1) GetBlockHeader(*bc.Hash) (*types.BlockHeader, error)          { return nil, nil }
func (s *mockStore1) GetStoreStatus() *BlockStoreState                             { return nil }
func (s *mockStore1) GetTransactionStatus(*bc.Hash) (*bc.TransactionStatus, error) { return nil, nil }
func (s *mockStore1) GetTransactionsUtxo(utxoView *state.UtxoViewpoint, tx []*bc.Tx) error {
	// TODO:
	for _, hash := range testTxs[2].SpentOutputIDs {
		utxoView.Entries[hash] = &storage.UtxoEntry{Type: storage.NormalUTXOType, Spent: false}
	}
	return nil
}
func (s *mockStore1) GetUtxo(*bc.Hash) (*storage.UtxoEntry, error)        { return nil, nil }
func (s *mockStore1) GetVoteResult(uint64) (*state.VoteResult, error)     { return nil, nil }
func (s *mockStore1) GetMainChainHash(uint64) (*bc.Hash, error)           { return nil, nil }
func (s *mockStore1) GetBlockHashesByHeight(uint64) ([]*bc.Hash, error)   { return nil, nil }
func (s *mockStore1) SaveBlock(*types.Block, *bc.TransactionStatus) error { return nil }
func (s *mockStore1) SaveBlockHeader(*types.BlockHeader) error            { return nil }
func (s *mockStore1) SaveChainStatus(*types.BlockHeader, *types.BlockHeader, []*types.BlockHeader, *state.UtxoViewpoint, []*state.VoteResult) error {
	return nil
}

func TestProcessTransaction(t *testing.T) {
	txPool := &TxPool{
		pool:            make(map[bc.Hash]*TxDesc),
		utxo:            make(map[bc.Hash]*types.Tx),
		orphans:         make(map[bc.Hash]*orphanTx),
		orphansByPrev:   make(map[bc.Hash]map[bc.Hash]*orphanTx),
		store:           &mockStore1{},
		eventDispatcher: event.NewDispatcher(),
	}
	cases := []struct {
		want  *TxPool
		addTx *TxDesc
	}{
		//Dust tx
		{
			want: &TxPool{},
			addTx: &TxDesc{
				Tx:         testTxs[3],
				StatusFail: false,
			},
		},
		//Dust tx
		{
			want: &TxPool{},
			addTx: &TxDesc{
				Tx:         testTxs[4],
				StatusFail: false,
			},
		},
		//Dust tx
		{
			want: &TxPool{},
			addTx: &TxDesc{
				Tx:         testTxs[5],
				StatusFail: false,
			},
		},
		//Dust tx
		{
			want: &TxPool{},
			addTx: &TxDesc{
				Tx:         testTxs[6],
				StatusFail: false,
			},
		},
		//normal tx
		{
			want: &TxPool{
				pool: map[bc.Hash]*TxDesc{
					testTxs[2].ID: {
						Tx:         testTxs[2],
						StatusFail: false,
						Weight:     150,
					},
				},
				utxo: map[bc.Hash]*types.Tx{
					*testTxs[2].ResultIds[0]: testTxs[2],
					*testTxs[2].ResultIds[1]: testTxs[2],
				},
			},
			addTx: &TxDesc{
				Tx:         testTxs[2],
				StatusFail: false,
			},
		},
	}

	for i, c := range cases {
		txPool.ProcessTransaction(c.addTx.Tx, c.addTx.StatusFail, 0, 0)
		for _, txD := range txPool.pool {
			txD.Added = time.Time{}
		}
		for _, txD := range txPool.orphans {
			txD.Added = time.Time{}
			txD.expiration = time.Time{}
		}

		if !testutil.DeepEqual(txPool.pool, c.want.pool) {
			t.Errorf("case %d: test ProcessTransaction pool mismatch got %s want %s", i, spew.Sdump(txPool.pool), spew.Sdump(c.want.pool))
		}
		if !testutil.DeepEqual(txPool.utxo, c.want.utxo) {
			t.Errorf("case %d: test ProcessTransaction utxo mismatch got %s want %s", i, spew.Sdump(txPool.utxo), spew.Sdump(c.want.utxo))
		}
		if !testutil.DeepEqual(txPool.orphans, c.want.orphans) {
			t.Errorf("case %d: test ProcessTransaction orphans mismatch got %s want %s", i, spew.Sdump(txPool.orphans), spew.Sdump(c.want.orphans))
		}
		if !testutil.DeepEqual(txPool.orphansByPrev, c.want.orphansByPrev) {
			t.Errorf("case %d: test ProcessTransaction orphansByPrev mismatch got %s want %s", i, spew.Sdump(txPool.orphansByPrev), spew.Sdump(c.want.orphansByPrev))
		}
	}
}
