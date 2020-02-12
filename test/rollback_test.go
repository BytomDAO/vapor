package test

import (
	"os"
	"testing"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/database"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
	"github.com/bytom/vapor/testutil"
)

const (
	InitBlockNum   = 1 // 初始化用的block数量
	appendBlockNum = 2 // 用于回滚的N的数量
)

func TestRollback(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	// init database
	store := database.NewStore(testDB)
	coinbaseTxData := &types.TxData{
		Version: 1,
		Inputs: []*types.TxInput{
			types.NewCoinbaseInput([]byte("Information is power. -- Jan/11/2013. Computing is power. -- Apr/24/2018.")),
		},
		Outputs: []*types.TxOutput{
			types.NewVoteOutput(*consensus.BTMAssetID, uint64(10000), []byte{0x51}, []byte{0x51}),
		},
	}

	initBlockHeaderArray := []*types.BlockHeader{}
	initBlockArray := []*types.Block{}

	appendBlockHeaderArray := []*types.BlockHeader{}
	appendBlockArray := []*types.Block{}

	coinbaseTx := types.NewTx(*coinbaseTxData)
	txs := []*bc.Tx{coinbaseTx.Tx}
	merkleRoot, _ := types.TxMerkleRoot(txs)
	txStatus := &bc.TransactionStatus{
		VerifyStatus: []*bc.TxVerifyResult{
			{StatusFail: false},
		},
	}
	txStatusHash, _ := types.TxStatusMerkleRoot([]*bc.TxVerifyResult{
		{StatusFail: false},
	})

	mainBlock := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Timestamp: uint64(1528945000),
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				TransactionStatusHash:  txStatusHash,
			},
		},
	}
	if err := store.SaveBlock(mainBlock, txStatus); err != nil {
		t.Fatal(err)
	}

	initBlockHeaderArray = append(initBlockHeaderArray, &mainBlock.BlockHeader)
	for i := 1; i <= InitBlockNum; i++ {
		mainBlock := &types.Block{
			BlockHeader: types.BlockHeader{
				PreviousBlockHash: mainBlock.Hash(),
				Version:           1,
				Height:            uint64(i),
				Timestamp:         uint64(1528945000 + i),
				BlockCommitment: types.BlockCommitment{
					TransactionsMerkleRoot: merkleRoot,
					TransactionStatusHash:  txStatusHash,
				},
			},
		}
		initBlockHeaderArray = append(initBlockHeaderArray, &mainBlock.BlockHeader)
		initBlockArray = append(initBlockArray, mainBlock)

		if err := store.SaveBlock(mainBlock, txStatus); err != nil {
			t.Fatal(err)
		}
	}

	for i := 1; i <= appendBlockNum; i++ {
		mainBlock := &types.Block{
			BlockHeader: types.BlockHeader{
				PreviousBlockHash: mainBlock.Hash(),
				Version:           1,
				Height:            uint64(i + InitBlockNum),
				Timestamp:         uint64(1528945000 + i + InitBlockNum),
				BlockCommitment: types.BlockCommitment{
					TransactionsMerkleRoot: merkleRoot,
					TransactionStatusHash:  txStatusHash,
				},
			},
		}
		appendBlockHeaderArray = append(appendBlockHeaderArray, &mainBlock.BlockHeader)
		appendBlockArray = append(appendBlockArray, mainBlock)

		if err := store.SaveBlock(mainBlock, txStatus); err != nil {
			t.Fatal(err)
		}
	}

	bestBlockHeader := initBlockHeaderArray[len(initBlockHeaderArray)-1]
	irrBlockHeader := bestBlockHeader

	if err := store.SaveChainStatus(bestBlockHeader, irrBlockHeader, initBlockHeaderArray, &state.UtxoViewpoint{}, []*state.ConsensusResult{}); err != nil {
		t.Fatal(err)
	}

	// run chain
	chain, store, _, err := MockChain(testDB)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(appendBlockArray); i++ {
		mainBlock := appendBlockArray[i]
		chain.ProcessBlock(mainBlock)
	}

	// rollback chain
	chain.Rollback(InitBlockNum)

	// compare status , and check
	for i := 0; i < len(initBlockArray); i++ {
		block := initBlockArray[i]
		blockHash := block.Hash()

		gotBlock, err := store.GetBlock(&blockHash)
		if err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(gotBlock, block) {
			t.Errorf("check 1: block mismatch: have %x, want %x", gotBlock, block)
		}
	}

	for i := 0; i < len(appendBlockArray); i++ {
		block := appendBlockArray[i]
		blockHash := block.Hash()

		if getBlock, err := store.GetBlock(&blockHash); err == nil {
			t.Errorf("check2: block exist :%v", getBlock)
		}
	}

}

// func TestRollbackBlockOld(t *testing.T) {
// 	testDB := dbm.NewDB("testdb", "leveldb", "temp")
// 	defer func() {
// 		testDB.Close()
// 		os.RemoveAll("temp")
// 	}()

// 	chain, store, _, err := MockChain(testDB)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	coinbaseTxData := &types.TxData{
// 		Version: 1,
// 		Inputs: []*types.TxInput{
// 			types.NewCoinbaseInput([]byte("Information is power. -- Jan/11/2013. Computing is power. -- Apr/24/2018.")),
// 		},
// 		Outputs: []*types.TxOutput{
// 			types.NewVoteOutput(*consensus.BTMAssetID, uint64(10000), []byte{0x51}, []byte{0x51}),
// 		},
// 	}

// 	coinbaseTx := types.NewTx(*coinbaseTxData)
// 	txs := []*bc.Tx{coinbaseTx.Tx}
// 	merkleRoot, _ := types.TxMerkleRoot(txs)

// 	initBlockHeader := &types.BlockHeader{
// 		Height:  0,
// 		Version: 1,
// 	}
// 	if err := store.SaveBlockHeader(initBlockHeader); err != nil {
// 		t.Fatal(err)
// 	}

// 	blockHash := initBlockHeader.Hash() //Hash: bc.Hash{V0: 0, V1: 1, V2: 2, V3: 3}
// 	view := &state.UtxoViewpoint{
// 		Entries: map[bc.Hash]*storage.UtxoEntry{
// 			bc.Hash{V0: 1, V1: 2, V2: 3, V3: 4}: &storage.UtxoEntry{Type: storage.NormalUTXOType, BlockHeight: 100, Spent: false},
// 			bc.Hash{V0: 1, V1: 2, V2: 3, V3: 4}: &storage.UtxoEntry{Type: storage.CoinbaseUTXOType, BlockHeight: 100, Spent: true},
// 			bc.Hash{V0: 1, V1: 1, V2: 3, V3: 4}: &storage.UtxoEntry{Type: storage.NormalUTXOType, BlockHeight: 100, Spent: true},
// 			bc.Hash{V0: 1, V1: 1, V2: 3, V3: 5}: &storage.UtxoEntry{Type: storage.CrosschainUTXOType, BlockHeight: 100, Spent: false},
// 			bc.Hash{V0: 1, V1: 1, V2: 3, V3: 6}: &storage.UtxoEntry{Type: storage.CrosschainUTXOType, BlockHeight: 100, Spent: true},
// 			bc.Hash{V0: 1, V1: 3, V2: 3, V3: 7}: &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 100, Spent: false},
// 			bc.Hash{V0: 1, V1: 3, V2: 3, V3: 7}: &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 100, Spent: true},
// 		},
// 	}
// 	if err := store.SaveChainStatus(initBlockHeader, initBlockHeader, []*types.BlockHeader{initBlockHeader}, view, []*state.ConsensusResult{}); err != nil {
// 		t.Fatal(err)
// 	}

// 	expectStatus := &protocol.BlockStoreState{Height: initBlockHeader.Height, Hash: &blockHash, IrreversibleHeight: initBlockHeader.Height, IrreversibleHash: &blockHash}
// 	if !testutil.DeepEqual(store.GetStoreStatus(), expectStatus) {
// 		t.Errorf("got block status:%v, expect block status:%v", store.GetStoreStatus(), expectStatus)
// 	}

// 	txStatus := &bc.TransactionStatus{
// 		VerifyStatus: []*bc.TxVerifyResult{
// 			{StatusFail: false},
// 		},
// 	}
// 	txStatusHash, _ := types.TxStatusMerkleRoot([]*bc.TxVerifyResult{
// 		{StatusFail: false},
// 	})

// 	mainBlock := &types.Block{
// 		BlockHeader: types.BlockHeader{
// 			Version:   initBlockHeader.Version,
// 			Height:    initBlockHeader.Height,
// 			Timestamp: initBlockHeader.Timestamp,
// 			BlockCommitment: types.BlockCommitment{
// 				TransactionsMerkleRoot: merkleRoot,
// 				TransactionStatusHash:  txStatusHash,
// 			},
// 		},
// 	}

// 	if err := store.SaveBlock(mainBlock, txStatus); err != nil {
// 		t.Fatal(err)
// 	}

// 	blocks := []*types.Block{}
// 	mainChainBlockHeader := initBlockHeader
// 	for i := 0; i < 7; i++ {
// 		mainChainBlockHeader = &types.BlockHeader{
// 			PreviousBlockHash: mainChainBlockHeader.Hash(),
// 			Height:            uint64(i + 1),
// 		}
// 		mainBlock = &types.Block{
// 			BlockHeader: types.BlockHeader{
// 				Version:   mainChainBlockHeader.Version,
// 				Height:    mainChainBlockHeader.Height,
// 				Timestamp: mainChainBlockHeader.Timestamp,
// 				BlockCommitment: types.BlockCommitment{
// 					TransactionsMerkleRoot: merkleRoot,
// 					TransactionStatusHash:  txStatusHash,
// 				},
// 			},
// 		}

// 		blocks = append(blocks, mainBlock)

// 		if err := store.SaveBlockHeader(mainChainBlockHeader); err != nil {
// 			t.Fatal(err)
// 		}

// 		if err := store.SaveBlock(mainBlock, txStatus); err != nil {
// 			t.Fatal(err)
// 		}
// 	}

// 	if err := chain.Rollback(0); err != nil {
// 		t.Fatal(err)
// 	}

// 	for i := 0; i < len(blocks); i++ {
// 		block := blocks[i]
// 		blockHash := block.Hash()

// 		if getBlock, err := store.GetBlock(&blockHash); err == nil {
// 			t.Errorf("check2: block exist :%v", getBlock)
// 		}
// 	}

// 	fmt.Println("abcd!")
// }
