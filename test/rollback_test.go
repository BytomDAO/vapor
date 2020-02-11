package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/bytom/vapor/consensus"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
)

func TestRollbackBlock(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	chain, store, _, err := MockChain(testDB)
	if err != nil {
		t.Fatal(err)
	}

	coinbaseTxData := &types.TxData{
		Version: 1,
		Inputs: []*types.TxInput{
			types.NewCoinbaseInput([]byte("Information is power. -- Jan/11/2013. Computing is power. -- Apr/24/2018.")),
		},
		Outputs: []*types.TxOutput{
			types.NewVoteOutput(*consensus.BTMAssetID, uint64(10000), []byte{0x51}, []byte{0x51}),
		},
	}

	coinbaseTx := types.NewTx(*coinbaseTxData)
	txs := []*bc.Tx{coinbaseTx.Tx}
	merkleRoot, _ := types.TxMerkleRoot(txs)

	initBlockHeader := &types.BlockHeader{
		Height:  0,
		Version: 1,
	}
	if err := store.SaveBlockHeader(initBlockHeader); err != nil {
		t.Fatal(err)
	}

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
			Version:   initBlockHeader.Version,
			Height:    initBlockHeader.Height,
			Timestamp: initBlockHeader.Timestamp,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				TransactionStatusHash:  txStatusHash,
			},
		},
	}

	if err := store.SaveBlock(mainBlock, txStatus); err != nil {
		t.Fatal(err)
	}

	blocks := []*types.Block{}
	mainChainBlockHeader := initBlockHeader
	for i := 0; i < 7; i++ {
		mainChainBlockHeader = &types.BlockHeader{
			PreviousBlockHash: mainChainBlockHeader.Hash(),
			Height:            uint64(i + 1),
		}
		mainBlock = &types.Block{
			BlockHeader: types.BlockHeader{
				Version:   mainChainBlockHeader.Version,
				Height:    mainChainBlockHeader.Height,
				Timestamp: mainChainBlockHeader.Timestamp,
				BlockCommitment: types.BlockCommitment{
					TransactionsMerkleRoot: merkleRoot,
					TransactionStatusHash:  txStatusHash,
				},
			},
		}

		chain.SetBestBlockHeader(mainChainBlockHeader)
		blocks = append(blocks, mainBlock)

		if err := store.SaveBlockHeader(mainChainBlockHeader); err != nil {
			t.Fatal(err)
		}

		if err := store.SaveBlock(mainBlock, txStatus); err != nil {
			t.Fatal(err)
		}
	}

	if err := chain.Rollback(0); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(blocks); i++ {
		block := blocks[i]
		blockHash := block.Hash()

		if getBlock, err := store.GetBlock(&blockHash); err == nil {
			t.Errorf("check2: block exist :%v", getBlock)
		}
	}

	fmt.Println("abcd!")
}
