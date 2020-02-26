package protocol

import (
	"sync"
	"testing"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
	"github.com/bytom/vapor/testutil"
)

type rProtocoler struct {
	startHeight uint64
}

func newRProtocoler(height uint64) *rProtocoler {
	return &rProtocoler{
		startHeight: height,
	}
}

func (p *rProtocoler) Name() string        { return "Mock" }
func (p *rProtocoler) StartHeight() uint64 { return p.startHeight }
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

func TestRollbackMock(t *testing.T) {
	cases := []struct {
		desc                      string
		protocolerStartHeight     uint64
		bestBlockHeader           *types.BlockHeader
		lastIrrBlockHeader        *types.BlockHeader
		storedBlocks              []*types.Block
		storedConsensusResult     []*state.ConsensusResult
		beforeBestConsensusResult *state.ConsensusResult
		expectBestBlockHeader     *types.BlockHeader
		expectLastIrrBlockHeader  *types.BlockHeader
		expectBestConsensusResult *state.ConsensusResult
		targetHeight              uint64
	}{
		{
			desc:                  "first case",
			protocolerStartHeight: 0,
			bestBlockHeader: &types.BlockHeader{
				Height:            1,
				PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
			},
			expectBestBlockHeader: &types.BlockHeader{
				Height: 0,
			},
			lastIrrBlockHeader: &types.BlockHeader{
				Height:            1,
				PreviousBlockHash: testutil.MustDecodeHash("39dee75363127a2857f554d2ad2706eb876407a2e09fbe0338683ca4ad4c2f90"),
			},
			expectLastIrrBlockHeader: &types.BlockHeader{
				Height: 0,
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
			storedConsensusResult: []*state.ConsensusResult{
				{
					Seq: 1,
					NumOfVote: map[string]uint64{
						"b7f463446a31b3792cd168d52b7a89b3657bca3e25d6854db1488c389ab6fc8d538155c25c1ee6975cc7def19710908c7d9b7463ca34a22058b456b45e498db9": 100002000,
						"36695997983028c279c3360ca345a90e3af1f9e3df2506119fca31cdc844be31630f9a421f4d1658e15d67a15ce29c36332dd45020d2a0147fcce4949ccd9a67": 100002000,
					},
					BlockHash:      testutil.MustDecodeHash("52463075c66259098f2a1fa711288cf3b866d7c57b4a7a78cd22a1dcd69a0514"),
					BlockHeight:    1,
					CoinbaseReward: map[string]uint64{"0001": consensus.BlockSubsidy(1)},
				},
				{
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
			expectBestConsensusResult: &state.ConsensusResult{
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
		mockStore := newDummyStore()
		mockProtocoler := newRProtocoler(c.protocolerStartHeight)

		for _, block := range c.storedBlocks {
			newTrans := []*types.Tx{}
			status := bc.NewTransactionStatus()
			for index, tx := range block.Transactions {
				status.SetStatus(index, false)
				tx := &types.Tx{TxData: tx.TxData, Tx: types.MapTx(&tx.TxData)}
				newTrans = append(newTrans, tx)
			}
			block.Transactions = newTrans
			mockStore.SaveBlock(block, status)
		}

		for _, consensusResult := range c.storedConsensusResult {
			mockStore.SetConsensusResult(consensusResult)
		}

		chain := &Chain{
			store:              mockStore,
			subProtocols:       []Protocoler{mockProtocoler},
			bestBlockHeader:    c.bestBlockHeader,
			lastIrrBlockHeader: c.lastIrrBlockHeader,
		}

		chain.cond.L = new(sync.Mutex)

		if err := chain.Rollback(c.targetHeight); err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(chain.lastIrrBlockHeader, c.expectLastIrrBlockHeader) {
			t.Fatalf("lastIrrBlockHeader is not right!")
		}

		if !testutil.DeepEqual(chain.bestBlockHeader, c.expectBestBlockHeader) {
			t.Fatalf("expectBestBlockHeader is not right!")
		}

		nowConsensusResult, err := chain.getBestConsensusResult()
		if err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(nowConsensusResult, c.expectBestConsensusResult) {
			t.Fatalf("expectBestConsensusResult is not right!")
		}
	}
}
