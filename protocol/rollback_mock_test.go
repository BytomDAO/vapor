package protocol

import (
	"fmt"
	"sync"
	"testing"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/database/storage"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
	"github.com/bytom/vapor/testutil"
)

type rProtocoler struct {
}

func newRProtocoler() *rProtocoler {
	return &rProtocoler{}
}

func (p *rProtocoler) Name() string        { return "Mock" }
func (p *rProtocoler) StartHeight() uint64 { return 10 }
func (p *rProtocoler) BeforeProposalBlock(txs []*types.Tx, nodeProgram []byte, blockHeight uint64, gasLeft int64, isTimeout func() bool) ([]*types.Tx, error) {
	return nil, nil
}
func (p *rProtocoler) ChainStatus() (uint64, *bc.Hash, error) { return 0, nil, nil }
func (p *rProtocoler) ValidateBlock(block *types.Block, verifyResults []*bc.TxVerifyResult) error {
	return nil
}
func (p *rProtocoler) ValidateTxs(txs []*types.Tx, verifyResults []*bc.TxVerifyResult) error {
	return nil
}
func (p *rProtocoler) ValidateTx(tx *types.Tx, verifyResult *bc.TxVerifyResult) error { return nil }
func (p *rProtocoler) ApplyBlock(block *types.Block) error                            { return nil }
func (p *rProtocoler) DetachBlock(block *types.Block) error                           { return nil }

type rStore struct {
	blockHeaders      map[string]*types.BlockHeader
	blocks            map[string]*types.Block
	consensusResults  map[uint64]*state.ConsensusResult
	mainChainHash     map[uint64]*bc.Hash
	transactionStatus map[string]*bc.TransactionStatus
}

// newRStore create new AccountStore.
func newRStore() *rStore {
	return &rStore{
		blockHeaders:      make(map[string]*types.BlockHeader),
		blocks:            make(map[string]*types.Block),
		consensusResults:  make(map[uint64]*state.ConsensusResult),
		mainChainHash:     make(map[uint64]*bc.Hash),
		transactionStatus: make(map[string]*bc.TransactionStatus),
	}
}

func (s *rStore) BlockExist(hash *bc.Hash) bool { return false }
func (s *rStore) GetBlock(hash *bc.Hash) (*types.Block, error) {
	return s.blocks[hash.String()], nil
}
func (s *rStore) GetBlockHeader(hash *bc.Hash) (*types.BlockHeader, error) {
	return s.blockHeaders[hash.String()], nil
}
func (s *rStore) GetStoreStatus() *BlockStoreState { return nil }
func (s *rStore) GetTransactionStatus(hash *bc.Hash) (*bc.TransactionStatus, error) {
	return s.transactionStatus[hash.String()], nil
}
func (s *rStore) GetTransactionsUtxo(*state.UtxoViewpoint, []*bc.Tx) error { return nil }
func (s *rStore) GetUtxo(*bc.Hash) (*storage.UtxoEntry, error)             { return nil, nil }
func (s *rStore) GetMainChainHash(height uint64) (*bc.Hash, error) {
	return s.mainChainHash[height], nil
}
func (s *rStore) GetBlockHashesByHeight(uint64) ([]*bc.Hash, error) { return nil, nil }
func (s *rStore) DeleteConsensusResult(seq uint64) error {
	delete(s.consensusResults, seq)
	return nil
}
func (s *rStore) DeleteBlock(*types.Block) error { return nil }

func (s *rStore) GetConsensusResult(seq uint64) (*state.ConsensusResult, error) {
	return s.consensusResults[seq], nil
}
func (s *rStore) SetConsensusResult(consensusResult *state.ConsensusResult) {
	s.consensusResults[consensusResult.Seq] = consensusResult
}

func (s *rStore) SaveBlock(block *types.Block, status *bc.TransactionStatus) error {
	hash := block.Hash()
	fmt.Println("has save block hash:", hash.String(), block.Height)
	s.transactionStatus[hash.String()] = status
	s.mainChainHash[block.Height] = &hash
	s.blockHeaders[hash.String()] = &block.BlockHeader
	s.blocks[hash.String()] = block
	return nil
}
func (s *rStore) SaveBlockHeader(header *types.BlockHeader) error {
	hash := header.Hash()
	fmt.Println("has save hash:", hash.String(), header.Height)
	s.blockHeaders[hash.String()] = header
	return nil
}
func (s *rStore) SaveChainStatus(*types.BlockHeader, *types.BlockHeader, []*types.BlockHeader, *state.UtxoViewpoint, []*state.ConsensusResult) error {
	return nil
}

func TestRollbackMock(t *testing.T) {
	fmt.Println("\n\n\nTestRollbackMock")
	cases := []struct {
		desc                      string
		bestBlockHeader           *types.BlockHeader
		storedBlocks              []*types.Block
		beforeBestConsensusResult *state.ConsensusResult
		afterBestConsensusResult  *state.ConsensusResult
		targetHeight              uint64
	}{
		{
			desc: "first case",
			bestBlockHeader: &types.BlockHeader{
				Height:            1,
				PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
			},
			targetHeight: 0,

			storedBlocks: []*types.Block{
				{
					BlockHeader: types.BlockHeader{
						Height: 0,
					},
					Transactions: []*types.Tx{
						{
							TxData: types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 1000, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 1000, []byte{0, 1}, testutil.MustDecodeHexString("36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67")),
								},
							},
						},
					},
				},
				{
					BlockHeader: types.BlockHeader{
						Height:            1,
						PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
					},
					Transactions: []*types.Tx{
						{
							TxData: types.TxData{
								Inputs: []*types.TxInput{
									types.NewSpendInput(nil, bc.NewHash([32]byte{0, 1}), *consensus.BTMAssetID, 2000, 0, []byte{0, 1}),
								},
								Outputs: []*types.TxOutput{
									types.NewVoteOutput(*consensus.BTMAssetID, 2000, []byte{0, 1}, testutil.MustDecodeHexString("b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9")),
								},
							},
						},
					},
				},
			},
			beforeBestConsensusResult: &state.ConsensusResult{
				Seq: 1,
				NumOfVote: map[string]uint64{
					"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100002000,
					"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 100002000,
				},
				BlockHash:      testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
				BlockHeight:    1,
				CoinbaseReward: map[string]uint64{"0001": consensus.BlockSubsidy(1)},
			},
			afterBestConsensusResult: &state.ConsensusResult{
				Seq: 0,
				NumOfVote: map[string]uint64{
					"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100000000,
					"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 100002000,
				},
				BlockHash:      testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
				BlockHeight:    0,
				CoinbaseReward: map[string]uint64{"0001": 0},
			},
		},
	}

	for _, c := range cases {
		mockStore := newRStore()
		mockProtocoler := newRProtocoler()

		// for _, header := range c.storedBlockHeaders {
		// 	mockStore.SaveBlockHeader(header)
		// }

		for _, block := range c.storedBlocks {
			status := bc.NewTransactionStatus()
			for index, _ := range block.Transactions {
				status.SetStatus(index, false)
			}
			fmt.Println("block", block)
			fmt.Println("what start:", block.Transactions, block.Transactions[0])
			mockStore.SaveBlock(block, status)
		}

		mockStore.SetConsensusResult(c.beforeBestConsensusResult)

		chain := &Chain{
			store:           mockStore,
			subProtocols:    []Protocoler{mockProtocoler},
			bestBlockHeader: c.bestBlockHeader,
		}

		chain.cond.L = new(sync.Mutex)

		if err := chain.Rollback(c.targetHeight); err != nil {
			t.Fatal(err)
		}

	}
}
