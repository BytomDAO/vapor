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
func (s *mStore) GetVoteResult(uint64) (*state.VoteResult, error)              { return nil, nil }
func (s *mStore) GetMainChainHash(uint64) (*bc.Hash, error)                    { return nil, nil }
func (s *mStore) GetBlockHashesByHeight(uint64) ([]*bc.Hash, error)            { return nil, nil }
func (s *mStore) SaveBlock(*types.Block, *bc.TransactionStatus) error          { return nil }
func (s *mStore) SaveBlockHeader(blockHeader *types.BlockHeader) error {
	s.blockHeaders[blockHeader.Hash()] = blockHeader
	return nil
}
func (s *mStore) SaveChainStatus(*types.BlockHeader, *types.BlockHeader, []*types.BlockHeader, *state.UtxoViewpoint, []*state.VoteResult) error {
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

	getAttachBlockHeaders, getDetachBlockHeaders, _ := c.calcReorganizeChain(newChainBlockHeader, mainChainBlockHeader)
	if !testutil.DeepEqual(wantAttachBlockHeaders, getAttachBlockHeaders) {
		t.Errorf("attach headers want %v but get %v", wantAttachBlockHeaders, getAttachBlockHeaders)
	}

	if !testutil.DeepEqual(wantDetachBlockHeaders, getDetachBlockHeaders) {
		t.Errorf("detach headers want %v but get %v", wantDetachBlockHeaders, getDetachBlockHeaders)
	}
}
