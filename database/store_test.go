package database

import (
	"os"
	"testing"

	"github.com/vapor/config"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/database/storage"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
	"github.com/vapor/testutil"
)

func TestLoadBlockIndex(t *testing.T) {
	config.CommonConfig = config.DefaultConfig()
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	store := NewStore(testDB)
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	block := config.GenesisBlock()
	txStatus := bc.NewTransactionStatus()

	if err := store.SaveBlock(block, txStatus); err != nil {
		t.Fatal(err)
	}

	for block.Height <= 128 {
		preHash := block.Hash()
		block.PreviousBlockHash = preHash
		block.Height++
		if err := store.SaveBlock(block, txStatus); err != nil {
			t.Fatal(err)
		}

		if block.Height%32 != 0 {
			continue
		}

		for i := uint64(0); i < block.Height/32; i++ {
			block.Version++
			if err := store.SaveBlock(block, txStatus); err != nil {
				t.Fatal(err)
			}
		}
	}

	if _, err := store.LoadBlockIndex(128); err != nil {
		t.Fatal(err)
	}
}

func TestLoadBlockIndexBestHeight(t *testing.T) {
	cases := []struct {
		blockBestHeight uint64
		stateBestHeight uint64
	}{
		{
			blockBestHeight: 100,
			stateBestHeight: 90,
		},
		{
			blockBestHeight: 100,
			stateBestHeight: 0,
		},
		{
			blockBestHeight: 100,
			stateBestHeight: 100,
		},
	}

	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()
	store := NewStore(testDB)
	var savedBlocks []types.Block

	for _, c := range cases {
		block := config.GenesisBlock()
		txStatus := bc.NewTransactionStatus()

		for i := uint64(0); i < c.blockBestHeight; i++ {
			if err := store.SaveBlock(block, txStatus); err != nil {
				t.Fatal(err)
			}

			savedBlocks = append(savedBlocks, *block)
			block.PreviousBlockHash = block.Hash()
			block.Height++
		}

		index, err := store.LoadBlockIndex(c.stateBestHeight)
		if err != nil {
			t.Fatal(err)
		}

		for _, block := range savedBlocks {
			blockHash := block.Hash()
			if block.Height <= c.stateBestHeight != index.BlockExist(&blockHash) {
				t.Errorf("Error in load block index")
			}
		}
	}
}

func TestLoadBlockIndexEquals(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	store := NewStore(testDB)
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	block := config.GenesisBlock()
	txStatus := bc.NewTransactionStatus()
	expectBlockIndex := state.NewBlockIndex()
	var parent *state.BlockNode

	for block.Height <= 100 {
		if err := store.SaveBlock(block, txStatus); err != nil {
			t.Fatal(err)
		}

		if block.Height != 0 {
			parent = expectBlockIndex.GetNode(&block.PreviousBlockHash)
		}

		node, err := state.NewBlockNode(&block.BlockHeader, parent)
		if err != nil {
			t.Fatal(err)
		}

		expectBlockIndex.AddNode(node)
		block.PreviousBlockHash = block.Hash()
		block.Height++
	}

	index, err := store.LoadBlockIndex(100)
	if err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(expectBlockIndex, index) {
		t.Errorf("got block index:%v, expect block index:%v", index, expectBlockIndex)
	}
}
func TestSaveChainStatus(t *testing.T) {
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	store := NewStore(testDB)

	node := &state.BlockNode{Height: 100, Hash: bc.Hash{V0: 0, V1: 1, V2: 2, V3: 3}}
	view := &state.UtxoViewpoint{
		Entries: map[bc.Hash]*storage.UtxoEntry{
			bc.Hash{V0: 1, V1: 2, V2: 3, V3: 4}: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 100, Spent: false},
			bc.Hash{V0: 1, V1: 2, V2: 3, V3: 4}: &storage.UtxoEntry{IsCoinBase: true, BlockHeight: 100, Spent: true},
			bc.Hash{V0: 1, V1: 1, V2: 3, V3: 4}: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 100, Spent: true},
		},
	}

	if err := store.SaveChainStatus(node, node, view, map[uint64]*state.VoteResult{}); err != nil {
		t.Fatal(err)
	}

	expectStatus := &protocol.BlockStoreState{Height: node.Height, Hash: &node.Hash, IrreversibleHeight: node.Height, IrreversibleHash: &node.Hash}
	if !testutil.DeepEqual(store.GetStoreStatus(), expectStatus) {
		t.Errorf("got block status:%v, expect block status:%v", store.GetStoreStatus(), expectStatus)
	}

	for hash, utxo := range view.Entries {
		if utxo.Spent && !utxo.IsCoinBase {
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

	block := config.GenesisBlock()
	status := &bc.TransactionStatus{VerifyStatus: []*bc.TxVerifyResult{{StatusFail: true}}}
	if err := store.SaveBlock(block, status); err != nil {
		t.Fatal(err)
	}

	blockHash := block.Hash()
	gotBlock, err := store.GetBlock(&blockHash)
	if err != nil {
		t.Fatal(err)
	}

	gotBlock.Transactions[0].Tx.SerializedSize = 0
	gotBlock.Transactions[0].SerializedSize = 0
	if !testutil.DeepEqual(block, gotBlock) {
		t.Errorf("got block:%v, expect block:%v", gotBlock, block)
	}

	gotStatus, err := store.GetTransactionStatus(&blockHash)
	if err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(status, gotStatus) {
		t.Errorf("got status:%v, expect status:%v", gotStatus, status)
	}

	data := store.db.Get(calcBlockHeaderKey(block.Height, &blockHash))
	gotBlockHeader := types.BlockHeader{}
	if err := gotBlockHeader.UnmarshalText(data); err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(block.BlockHeader, gotBlockHeader) {
		t.Errorf("got block header:%v, expect block header:%v", gotBlockHeader, block.BlockHeader)
	}
}
