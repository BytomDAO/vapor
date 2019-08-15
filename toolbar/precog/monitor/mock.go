package monitor

import (
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

type mockChain struct{}

func (m *mockChain) BestBlockHeader() *types.BlockHeader {
	return &types.BlockHeader{}
}

func (m *mockChain) BestBlockHeight() uint64 {
	return 0
}

func (m *mockChain) GetBlockByHash(*bc.Hash) (*types.Block, error) {
	return &types.Block{}, nil
}

func (m *mockChain) GetBlockByHeight(uint64) (*types.Block, error) {
	return &types.Block{}, nil
}

func (m *mockChain) GetHeaderByHash(*bc.Hash) (*types.BlockHeader, error) {
	return &types.BlockHeader{}, nil
}

func (m *mockChain) GetHeaderByHeight(uint64) (*types.BlockHeader, error) {
	return &types.BlockHeader{}, nil
}

func (m *mockChain) GetTransactionStatus(*bc.Hash) (*bc.TransactionStatus, error) {
	return &bc.TransactionStatus{}, nil
}

type mockTxPool struct{}
type mockFastSyncDB struct{}

func (m *mockFastSyncDB) Close() {
}

func (m *mockFastSyncDB) Delete([]byte) {
}

func (m *mockFastSyncDB) DeleteSync([]byte) {
}

func (m *mockFastSyncDB) Get([]byte) []byte {
}
