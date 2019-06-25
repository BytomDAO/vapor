package database

import (
	"os"
	"testing"

	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/database/storage"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
	"github.com/vapor/testutil"
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

	if err := store.SaveChainStatus(blockHeader, blockHeader, []*types.BlockHeader{blockHeader}, view, []*state.VoteResult{}); err != nil {
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
	block := mockGenesisBlock()
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

	data := store.db.Get(calcBlockHeaderKey(&blockHash))
	gotBlockHeader := types.BlockHeader{}
	if err := gotBlockHeader.UnmarshalText(data); err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(block.BlockHeader, gotBlockHeader) {
		t.Errorf("got block header:%v, expect block header:%v", gotBlockHeader, block.BlockHeader)
	}
}

func mockGenesisBlock() *types.Block {
	txData := types.TxData{
		Version: 1,
		Inputs: []*types.TxInput{
			types.NewCoinbaseInput([]byte("Information is power. -- Jan/11/2013. Computing is power. -- Apr/24/2018.")),
		},
		Outputs: []*types.TxOutput{
			types.NewVoteOutput(bc.AssetID{V0: 1}, uint64(10000), []byte{0x51}, []byte{0x51}),
		},
	}
	tx := types.NewTx(txData)
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)
	txStatusHash, _ := types.TxStatusMerkleRoot(txStatus.VerifyStatus)
	merkleRoot, _ := types.TxMerkleRoot([]*bc.Tx{tx.Tx})
	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Timestamp: 1528945000,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				TransactionStatusHash:  txStatusHash,
			},
		},
		Transactions: []*types.Tx{tx},
	}
	return block
}
