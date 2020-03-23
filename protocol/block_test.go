package protocol

import (
	"testing"

	"github.com/bytom/vapor/database/storage"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
	"github.com/bytom/vapor/testutil"
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
func (s *mStore) DeleteConsensusResult(seq uint64) error                       { return nil }
func (s *mStore) DeleteBlock(*types.Block) error                               { return nil }
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
