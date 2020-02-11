package database

import (
	"os"
	"testing"

	"github.com/bytom/vapor/consensus"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/database/storage"
	"github.com/bytom/vapor/protocol"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
	"github.com/bytom/vapor/testutil"
)

func TestSaveChainStatus(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	store := NewStore(testDB)

	blockHeader := &types.BlockHeader{Height: 100}
	blockHash := blockHeader.Hash() //Hash: bc.Hash{V0: 0, V1: 1, V2: 2, V3: 3}
	view := &state.UtxoViewpoint{
		Entries: map[bc.Hash]*storage.UtxoEntry{
			bc.Hash{V0: 1, V1: 2, V2: 3, V3: 4}: &storage.UtxoEntry{Type: storage.NormalUTXOType, BlockHeight: 100, Spent: false},
			bc.Hash{V0: 1, V1: 2, V2: 3, V3: 4}: &storage.UtxoEntry{Type: storage.CoinbaseUTXOType, BlockHeight: 100, Spent: true},
			bc.Hash{V0: 1, V1: 1, V2: 3, V3: 4}: &storage.UtxoEntry{Type: storage.NormalUTXOType, BlockHeight: 100, Spent: true},
			bc.Hash{V0: 1, V1: 1, V2: 3, V3: 5}: &storage.UtxoEntry{Type: storage.CrosschainUTXOType, BlockHeight: 100, Spent: false},
			bc.Hash{V0: 1, V1: 1, V2: 3, V3: 6}: &storage.UtxoEntry{Type: storage.CrosschainUTXOType, BlockHeight: 100, Spent: true},
			bc.Hash{V0: 1, V1: 3, V2: 3, V3: 7}: &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 100, Spent: false},
			bc.Hash{V0: 1, V1: 3, V2: 3, V3: 7}: &storage.UtxoEntry{Type: storage.VoteUTXOType, BlockHeight: 100, Spent: true},
		},
	}

	if err := store.SaveChainStatus(blockHeader, blockHeader, []*types.BlockHeader{blockHeader}, view, []*state.ConsensusResult{}); err != nil {
		t.Fatal(err)
	}

	expectStatus := &protocol.BlockStoreState{Height: blockHeader.Height, Hash: &blockHash, IrreversibleHeight: blockHeader.Height, IrreversibleHash: &blockHash}
	if !testutil.DeepEqual(store.GetStoreStatus(), expectStatus) {
		t.Errorf("got block status:%v, expect block status:%v", store.GetStoreStatus(), expectStatus)
	}

	for hash, utxo := range view.Entries {
		if (utxo.Type == storage.NormalUTXOType) && utxo.Spent {
			continue
		}
		if (utxo.Type == storage.CrosschainUTXOType) && (!utxo.Spent) {
			continue
		}
		if (utxo.Type == storage.VoteUTXOType) && (utxo.Spent) {
			continue
		}

		gotUtxo, err := store.GetUtxo(&hash)
		if err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(utxo, gotUtxo) {
			t.Errorf("got utxo entry:%v, expect utxo entry:%v", gotUtxo, utxo)
		}
	}
}

func TestSaveBlock(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	store := NewStore(testDB)
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

	cases := []struct {
		txData      []*types.TxData
		txStatus    *bc.TransactionStatus
		blockHeader *types.BlockHeader
	}{
		{
			txStatus: &bc.TransactionStatus{
				VerifyStatus: []*bc.TxVerifyResult{
					{StatusFail: true},
				},
			},
			blockHeader: &types.BlockHeader{
				Version:   uint64(1),
				Height:    uint64(1111),
				Timestamp: uint64(1528945000),
			},
		},
		{
			txStatus: &bc.TransactionStatus{
				VerifyStatus: []*bc.TxVerifyResult{
					{StatusFail: false},
				},
			},
			blockHeader: &types.BlockHeader{
				Version:   uint64(1),
				Height:    uint64(1111),
				Timestamp: uint64(1528945000),
			},
		},
		{
			txData: []*types.TxData{
				{
					Version: 1,
					Inputs: []*types.TxInput{
						types.NewSpendInput([][]byte{}, bc.NewHash([32]byte{}), *consensus.BTMAssetID, 100000000, 0, []byte{0x51}),
					},
					Outputs: []*types.TxOutput{
						types.NewVoteOutput(*consensus.BTMAssetID, uint64(10000), []byte{0x51}, []byte{0x51}),
					},
				},
			},
			txStatus: &bc.TransactionStatus{
				VerifyStatus: []*bc.TxVerifyResult{
					{StatusFail: true},
				},
			},
			blockHeader: &types.BlockHeader{
				Version:   uint64(1),
				Height:    uint64(1111),
				Timestamp: uint64(1528945000),
			},
		},
		{
			txData: []*types.TxData{
				{
					Version: 1,
					Inputs: []*types.TxInput{
						types.NewSpendInput([][]byte{}, bc.NewHash([32]byte{}), *consensus.BTMAssetID, 100000000, 0, []byte{0x51}),
					},
					Outputs: []*types.TxOutput{
						types.NewVoteOutput(*consensus.BTMAssetID, uint64(88888), []byte{0x51}, []byte{0x51}),
					},
				},
			},
			txStatus: &bc.TransactionStatus{
				VerifyStatus: []*bc.TxVerifyResult{
					{StatusFail: false},
				},
			},
			blockHeader: &types.BlockHeader{
				Version:   uint64(1),
				Height:    uint64(0),
				Timestamp: uint64(152894500000),
			},
		},
	}

	for i, c := range cases {
		txs := []*bc.Tx{coinbaseTx.Tx}
		for _, tx := range c.txData {
			t := types.NewTx(*tx)
			txs = append(txs, t.Tx)
		}
		merkleRoot, _ := types.TxMerkleRoot(txs)
		txStatusHash, _ := types.TxStatusMerkleRoot(c.txStatus.VerifyStatus)
		block := &types.Block{
			BlockHeader: types.BlockHeader{
				Version:   c.blockHeader.Version,
				Height:    c.blockHeader.Height,
				Timestamp: c.blockHeader.Timestamp,
				BlockCommitment: types.BlockCommitment{
					TransactionsMerkleRoot: merkleRoot,
					TransactionStatusHash:  txStatusHash,
				},
			},
		}

		if err := store.SaveBlock(block, c.txStatus); err != nil {
			t.Fatal(err)
		}

		blockHash := block.Hash()
		gotBlock, err := store.GetBlock(&blockHash)
		if err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(gotBlock, block) {
			t.Errorf("case %v: block mismatch: have %x, want %x", i, gotBlock, block)
		}

		gotStatus, err := store.GetTransactionStatus(&blockHash)
		if err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(gotStatus.VerifyStatus, c.txStatus.VerifyStatus) {
			t.Errorf("case %v: VerifyStatus mismatch: have %x, want %x", i, gotStatus.VerifyStatus, c.txStatus.VerifyStatus)
		}

		gotBlockHeader, err := store.GetBlockHeader(&blockHash)
		if err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(block.BlockHeader, *gotBlockHeader) {
			t.Errorf("got block header:%v, expect block header:%v", gotBlockHeader, block.BlockHeader)
		}
	}
}

func TestSaveBlockHeader(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	store := NewStore(testDB)

	cases := []struct {
		blockHeader *types.BlockHeader
	}{
		{
			blockHeader: &types.BlockHeader{
				Version:   uint64(1),
				Height:    uint64(1111),
				Timestamp: uint64(1528945000),
			},
		},
		{
			blockHeader: &types.BlockHeader{
				Version:           uint64(1),
				Height:            uint64(0),
				PreviousBlockHash: bc.NewHash([32]byte{0x3e, 0x94, 0x5d, 0x35, 0x70, 0x30, 0xd4, 0x3b, 0x3d, 0xe3, 0xdd, 0x80, 0x67, 0x29, 0x9a, 0x5e, 0x09, 0xf9, 0xfb, 0x2b, 0xad, 0x5f, 0x92, 0xc8, 0x69, 0xd1, 0x42, 0x39, 0x74, 0x9a, 0xd1, 0x1c}),
				Timestamp:         uint64(1563186936),
				BlockCommitment: types.BlockCommitment{
					TransactionsMerkleRoot: bc.NewHash([32]byte{0x3e, 0x94, 0x5d, 0x35, 0x70, 0x30, 0xd4, 0x3b, 0x3d, 0xe3, 0xdd, 0x80, 0x67, 0x29, 0x9a, 0x5e, 0x09, 0xf9, 0xfb, 0x2b, 0xad, 0x5f, 0x92, 0xc8, 0x69, 0xd1, 0x42, 0x39, 0x74, 0x9a, 0xd1, 0x1c}),
					TransactionStatusHash:  bc.NewHash([32]byte{0x3e, 0x94, 0x5d, 0x35, 0x70, 0x30, 0xd4, 0x3b, 0x3d, 0xe3, 0xdd, 0x80, 0x67, 0x29, 0x9a, 0x5e, 0x09, 0xf9, 0xfb, 0x2b, 0xad, 0x5f, 0x92, 0xc8, 0x69, 0xd1, 0x42, 0x39, 0x74, 0x9a, 0xd1, 0x1c}),
				},
				BlockWitness: types.BlockWitness{
					Witness: [][]byte{[]byte{0x3e, 0x94, 0x5d, 0x35}, []byte{0x3e, 0x94, 0x5d, 0x35}},
				},
			},
		},
		{
			blockHeader: &types.BlockHeader{
				Version:           uint64(1),
				Height:            uint64(8848),
				PreviousBlockHash: bc.NewHash([32]byte{0x3e, 0x94, 0x5d, 0x35, 0x70, 0x30, 0xd4, 0x3b, 0x3d, 0xe3, 0xdd, 0x80, 0x67, 0x29, 0x9a, 0x5e, 0x09, 0xf9, 0xfb, 0x2b, 0xad, 0x5f, 0x92, 0xc8, 0x69, 0xd1, 0x42, 0x39, 0x74, 0x9a, 0xd1, 0x1c}),
				Timestamp:         uint64(156318693600),
				BlockCommitment: types.BlockCommitment{
					TransactionsMerkleRoot: bc.NewHash([32]byte{0x3e, 0x94, 0x5d, 0x35, 0x70, 0x30, 0xd4, 0x3b, 0x3d, 0xe3, 0xdd, 0x80, 0x67, 0x29, 0x9a, 0x5e, 0x09, 0xf9, 0xfb, 0x2b, 0xad, 0x5f, 0x92, 0xc8, 0x69, 0xd1, 0x42, 0x39, 0x74, 0x9a, 0xd1, 0x1c}),
					TransactionStatusHash:  bc.NewHash([32]byte{0x3e, 0x94, 0x5d, 0x35, 0x70, 0x30, 0xd4, 0x3b, 0x3d, 0xe3, 0xdd, 0x80, 0x67, 0x29, 0x9a, 0x5e, 0x09, 0xf9, 0xfb, 0x2b, 0xad, 0x5f, 0x92, 0xc8, 0x69, 0xd1, 0x42, 0x39, 0x74, 0x9a, 0xd1, 0x1c}),
				},
				BlockWitness: types.BlockWitness{
					Witness: [][]byte{
						[]byte{0x3e, 0x94, 0x5d, 0x35},
						[]byte{0xdd, 0x80, 0x67, 0x29},
						[]byte{0xff, 0xff, 0xff, 0xff},
						[]byte{0x00, 0x01, 0x02, 0x03},
					},
				},
			},
		},
	}

	for i, c := range cases {
		if err := store.SaveBlockHeader(c.blockHeader); err != nil {
			t.Fatal(err)
		}

		blockHash := c.blockHeader.Hash()
		gotBlockHeader, err := store.GetBlockHeader(&blockHash)
		if err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(gotBlockHeader, c.blockHeader) {
			t.Errorf("case %v: block header mismatch: have %x, want %x", i, gotBlockHeader, c.blockHeader)
		}
	}
}

func TestDeleteBlock(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	store := NewStore(testDB)
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

	cases := []struct {
		blockHeaderData []*types.BlockHeader
		txStatusData    []*bc.TransactionStatus

		newBlockHeaderData []*types.BlockHeader
		newTxStatusData    []*bc.TransactionStatus
	}{
		{
			blockHeaderData: []*types.BlockHeader{
				{
					Version:   uint64(1),
					Height:    uint64(1),
					Timestamp: uint64(1528945000),
				},
				{
					Version:   uint64(1),
					Height:    uint64(2),
					Timestamp: uint64(1528945005),
				},
			},
			txStatusData: []*bc.TransactionStatus{
				{
					VerifyStatus: []*bc.TxVerifyResult{
						{StatusFail: false},
					},
				},
				{
					VerifyStatus: []*bc.TxVerifyResult{
						{StatusFail: false},
					},
				},
			},
			newBlockHeaderData: []*types.BlockHeader{
				{
					Version:   uint64(1),
					Height:    uint64(1),
					Timestamp: uint64(1528945030),
				},
				{
					Version:   uint64(1),
					Height:    uint64(2),
					Timestamp: uint64(1528945050),
				},
			},
			newTxStatusData: []*bc.TransactionStatus{
				{
					VerifyStatus: []*bc.TxVerifyResult{
						{StatusFail: false},
					},
				},
				{
					VerifyStatus: []*bc.TxVerifyResult{
						{StatusFail: false},
					},
				},
			},
		},
		{
			blockHeaderData: []*types.BlockHeader{
				{
					Version:   uint64(1),
					Height:    uint64(2),
					Timestamp: uint64(1528945005),
				},
			},
			txStatusData: []*bc.TransactionStatus{
				{
					VerifyStatus: []*bc.TxVerifyResult{
						{StatusFail: false},
					},
				},
			},
		},
	}

	for _, c := range cases {
		txs := []*bc.Tx{coinbaseTx.Tx}
		blockNum := len(c.blockHeaderData)
		storeBlockData := []*types.Block{}
		merkleRoot, _ := types.TxMerkleRoot(txs)
		for i := 0; i < blockNum; i++ {
			txStatusHash, _ := types.TxStatusMerkleRoot(c.txStatusData[i].VerifyStatus)
			block := &types.Block{
				BlockHeader: types.BlockHeader{
					Version:   c.blockHeaderData[i].Version,
					Height:    c.blockHeaderData[i].Height,
					Timestamp: c.blockHeaderData[i].Timestamp,
					BlockCommitment: types.BlockCommitment{
						TransactionsMerkleRoot: merkleRoot,
						TransactionStatusHash:  txStatusHash,
					},
				},
			}
			storeBlockData = append(storeBlockData, block)

			if err := store.SaveBlock(block, c.txStatusData[i]); err != nil {
				t.Fatal(err)
			}
		}

		for i := len(storeBlockData) - 1; i >= 0; i-- {
			block := storeBlockData[i]
			blockHash := block.Hash()
			if !store.BlockExist(&blockHash) {
				t.Errorf("check0: block not exist :%v, expect exists!", block)
			}

			if err := store.DeleteBlock(block); err != nil {
				t.Fatal(err)
			}

			if store.BlockExist(&blockHash) {
				t.Errorf("check1: block exist :%v, expect delete it!", block)
			}

			if getBlock, err := store.GetBlock(&blockHash); err == nil {
				t.Errorf("check2: block exist :%v", getBlock)
			}

		}

		for i := len(storeBlockData) - 1; i >= 0; i-- {
			block := storeBlockData[i]
			blockHash := block.Hash()

			if store.BlockExist(&blockHash) {
				t.Errorf("check3: block exist :%v, expect delete it!", block)
			}
		}

		newBlockNum := len(c.newBlockHeaderData)
		for i := 0; i < newBlockNum; i++ {
			txStatusHash, _ := types.TxStatusMerkleRoot(c.newTxStatusData[i].VerifyStatus)
			block := &types.Block{
				BlockHeader: types.BlockHeader{
					Version:   c.newBlockHeaderData[i].Version,
					Height:    c.newBlockHeaderData[i].Height,
					Timestamp: c.newBlockHeaderData[i].Timestamp,
					BlockCommitment: types.BlockCommitment{
						TransactionsMerkleRoot: merkleRoot,
						TransactionStatusHash:  txStatusHash,
					},
				},
			}

			if err := store.SaveBlock(block, c.newTxStatusData[i]); err != nil {
				t.Fatal(err)
			}

			blockHash := block.Hash()
			gotBlock, err := store.GetBlock(&blockHash)
			if err != nil {
				t.Fatal(err)
			}

			if !testutil.DeepEqual(gotBlock, block) {
				t.Errorf("case %v: block mismatch: have %x, want %x", i, gotBlock, block)
			}

			getBlockHeader, err := store.GetBlockHeader(&blockHash)
			if err != nil {
				t.Fatal(err)
			}

			if !testutil.DeepEqual(block.BlockHeader, *getBlockHeader) {
				t.Errorf("got block header:%v, expect block header:%v", getBlockHeader, block.BlockHeader)
			}
		}
	}
}
