package protocol

import (
	"testing"

	"github.com/vapor/database/storage"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
	"github.com/vapor/testutil"
)

type mStore struct {
	blockHeaders map[bc.Hash]*types.BlockHeader
}

func (s *mStore) BlockExist(hash *bc.Hash) bool           { return false }
func (s *mStore) GetBlock(*bc.Hash) (*types.Block, error) { return nil, nil }
func (s *mStore) GetBlockHeader(hash *bc.Hash) (*types.BlockHeader, error) {
	return s.blockHeaders[*hash], nil
}
func (s *mStore) GetStoreStatus() *BlockStoreState                             { return nil }
func (s *mStore) GetTransactionStatus(*bc.Hash) (*bc.TransactionStatus, error) { return nil, nil }
func (s *mStore) GetTransactionsUtxo(*state.UtxoViewpoint, []*bc.Tx) error     { return nil }
func (s *mStore) GetUtxo(*bc.Hash) (*storage.UtxoEntry, error)                 { return nil, nil }
func (s *mStore) GetConsensusResult(uint64) (*state.ConsensusResult, error)    { return nil, nil }
func (s *mStore) GetMainChainHash(uint64) (*bc.Hash, error)                    { return nil, nil }
func (s *mStore) GetBlockHashesByHeight(uint64) ([]*bc.Hash, error)            { return nil, nil }
func (s *mStore) SaveBlock(*types.Block, *bc.TransactionStatus) error          { return nil }
func (s *mStore) SaveBlockHeader(blockHeader *types.BlockHeader) error {
	s.blockHeaders[blockHeader.Hash()] = blockHeader
	return nil
}
func (s *mStore) SaveChainStatus(*types.BlockHeader, *types.BlockHeader, []*types.BlockHeader, *state.UtxoViewpoint, []*state.ConsensusResult) error {
	return nil
}

func TestCalcReorganizeChain(t *testing.T) {
	c := &Chain{
		store: &mStore{
			blockHeaders: make(map[bc.Hash]*types.BlockHeader),
		},
	}

	initBlockHeader := &types.BlockHeader{
		Height:  0,
		Version: 1,
	}
	c.store.SaveBlockHeader(initBlockHeader)

	var wantAttachBlockHeaders []*types.BlockHeader
	var wantDetachBlockHeaders []*types.BlockHeader
	mainChainBlockHeader := initBlockHeader
	newChainBlockHeader := initBlockHeader
	for i := 1; i <= 7; i++ {
		mainChainBlockHeader = &types.BlockHeader{
			PreviousBlockHash: mainChainBlockHeader.Hash(),
			Height:            uint64(i),
		}
		wantDetachBlockHeaders = append([]*types.BlockHeader{mainChainBlockHeader}, wantDetachBlockHeaders...)
		c.store.SaveBlockHeader(mainChainBlockHeader)
	}

	for i := 1; i <= 13; i++ {
		newChainBlockHeader = &types.BlockHeader{
			PreviousBlockHash: newChainBlockHeader.Hash(),
			Height:            uint64(i),
			Version:           1,
		}
		wantAttachBlockHeaders = append(wantAttachBlockHeaders, newChainBlockHeader)
		c.store.SaveBlockHeader(newChainBlockHeader)
	}

	// normal
	getAttachBlockHeaders, getDetachBlockHeaders, _ := c.calcReorganizeChain(newChainBlockHeader, mainChainBlockHeader)
	if !testutil.DeepEqual(wantAttachBlockHeaders, getAttachBlockHeaders) {
		t.Errorf("normal test: attach headers want %v but get %v", wantAttachBlockHeaders, getAttachBlockHeaders)
	}

	if !testutil.DeepEqual(wantDetachBlockHeaders, getDetachBlockHeaders) {
		t.Errorf("normal test: detach headers want %v but get %v", wantDetachBlockHeaders, getDetachBlockHeaders)
	}

	// detachBlockHeaders is empty
	forkChainBlockHeader := wantAttachBlockHeaders[7]
	wantAttachBlockHeaders = wantAttachBlockHeaders[8:]
	wantDetachBlockHeaders = []*types.BlockHeader{}
	getAttachBlockHeaders, getDetachBlockHeaders, _ = c.calcReorganizeChain(newChainBlockHeader, forkChainBlockHeader)
	if !testutil.DeepEqual(wantAttachBlockHeaders, getAttachBlockHeaders) {
		t.Errorf("detachBlockHeaders is empty test: attach headers want %v but get %v", wantAttachBlockHeaders, getAttachBlockHeaders)
	}

	if !testutil.DeepEqual(wantDetachBlockHeaders, getDetachBlockHeaders) {
		t.Errorf("detachBlockHeaders is empty test: detach headers want %v but get %v", wantDetachBlockHeaders, getDetachBlockHeaders)
	}
}

/*
func TestMockReorganizeChain(t *testing.T) {
	detachBlocks := []*types.Block{
		&types.Block{
			BlockHeader: types.BlockHeader{
				Height:            consensus.MainNetParams.RoundVoteBlockNums + 1,
				PreviousBlockHash: testutil.MustDecodeHash("4d08d14eea57211b41b5596828dbe9f34ca8ef40713042c57c6a3003b7147269"),
			},
			Transactions: []*types.Tx{
				&types.Tx{
					TxData: types.TxData{
						Inputs:  []*types.TxInput{types.NewCoinbaseInput([]byte{0x03})},
						Outputs: []*types.TxOutput{types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51})},
					},
				},
			},
		},
		&types.Block{
			BlockHeader: types.BlockHeader{
				Height:            consensus.MainNetParams.RoundVoteBlockNums,
				PreviousBlockHash: testutil.MustDecodeHash("4bcf2eed7829eab8220cee9e2e920c656aacb2de71619bad0536f6f7b9b00c2d"),
			},
			Transactions: []*types.Tx{
				&types.Tx{
					TxData: types.TxData{
						Inputs: []*types.TxInput{types.NewCoinbaseInput([]byte{0x02})},
						Outputs: []*types.TxOutput{
							types.NewIntraChainOutput(bc.AssetID{}, 0, []byte{0x51}),
							types.NewIntraChainOutput(bc.AssetID{}, consensus.BlockSubsidy(consensus.MainNetParams.RoundVoteBlockNums)+10000000, []byte{0x51}),
							types.NewIntraChainOutput(bc.AssetID{}, 30000000, []byte{0x53}),
							types.NewIntraChainOutput(bc.AssetID{}, 20000000, []byte{0x52}),
						},
					},
				},
			},
		},
		&types.Block{
			BlockHeader: types.BlockHeader{
				Height:            consensus.MainNetParams.RoundVoteBlockNums - 1,
				PreviousBlockHash: testutil.MustDecodeHash("48f81bf65aeb0b3bf8d65c82d46149fd835a62a7b23b7a781781808066d6c1e3"),
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
	}

	consensusResult := state.ConsensusResult{
		NumOfVote: map[string]uint64{},
		CoinbaseReward: map[string]uint64{
			"51": consensus.BlockSubsidy(consensus.MainNetParams.RoundVoteBlockNums),
		},
		BlockHash:   testutil.MustDecodeHash("391bffc3586e98067895bca12ffdbc82dff0fee1344e884b769727c1ac056ef7"),
		BlockHeight: consensus.MainNetParams.RoundVoteBlockNums + 2,
	}

	for _, detachBlock := range detachBlocks {
		if err := consensusResult.DetachBlock(detachBlock); err != nil {
			t.Fatal(err)
		}
	}

}
*/
