package netsync

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/vapor/consensus"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

var txs = []*types.Tx{
	types.NewTx(types.TxData{
		SerializedSize: uint64(52),
		Inputs:         []*types.TxInput{types.NewCoinbaseInput([]byte{0x01})},
		Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 5000, nil)},
	}),
	types.NewTx(types.TxData{
		SerializedSize: uint64(53),
		Inputs:         []*types.TxInput{types.NewCoinbaseInput([]byte{0x01, 0x02})},
		Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 5000, nil)},
	}),
	types.NewTx(types.TxData{
		SerializedSize: uint64(54),
		Inputs:         []*types.TxInput{types.NewCoinbaseInput([]byte{0x01, 0x02, 0x03})},
		Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 5000, nil)},
	}),
	types.NewTx(types.TxData{
		SerializedSize: uint64(54),
		Inputs:         []*types.TxInput{types.NewCoinbaseInput([]byte{0x01, 0x02, 0x03})},
		Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 2000, nil)},
	}),
	types.NewTx(types.TxData{
		SerializedSize: uint64(54),
		Inputs:         []*types.TxInput{types.NewCoinbaseInput([]byte{0x01, 0x02, 0x03})},
		Outputs:        []*types.TxOutput{types.NewIntraChainOutput(*consensus.BTMAssetID, 10000, nil)},
	}),
}

func TestTransactionMessage(t *testing.T) {
	wantStrings := [5]string{
		"{tx_size: 104, tx_hash: 8ec07dcaa0d8966ebf7d4483cff08cf2f91b7fd054d74b9b36e6701851b6987f}",
		"{tx_size: 106, tx_hash: b36797fadd33aa38794d4fef61a13214e17cd5441aee38f8cbdfe5a7863ee5ff}",
		"{tx_size: 108, tx_hash: bf93b35da3c58f26896657f7cab5034536b6ab70b3a32e1de07bff6f94318a94}",
		"{tx_size: 108, tx_hash: 82ba46e4d13ea302977564de1ea4053e0820f0c1296d404a05ed0adc3e3bac38}",
		"{tx_size: 108, tx_hash: fb6640b67c734e124e34ce65f52f877f192c36caad0830e312799cc337749bd9}",
	}
	for i, tx := range txs {
		txMsg, err := NewTransactionMessage(tx)
		if err != nil {
			t.Fatalf("create tx msg err:%s", err)
		}

		gotTx, err := txMsg.GetTransaction()
		if err != nil {
			t.Fatalf("get txs from txsMsg err:%s", err)
		}
		if !reflect.DeepEqual(*tx.Tx, *gotTx.Tx) {
			t.Errorf("txs msg test err: got %s\nwant %s", spew.Sdump(tx.Tx), spew.Sdump(gotTx.Tx))
		}

		if txMsg.String() != wantStrings[i] {
			t.Errorf("index:%d txs msg string test err. got:%s want:%s", i, txMsg.String(), wantStrings[i])
		}
	}
}

func TestTransactionsMessage(t *testing.T) {
	txsMsg, err := NewTransactionsMessage(txs)
	if err != nil {
		t.Fatalf("create txs msg err:%s", err)
	}

	gotTxs, err := txsMsg.GetTransactions()
	if err != nil {
		t.Fatalf("get txs from txsMsg err:%s", err)
	}

	if len(gotTxs) != len(txs) {
		t.Fatal("txs msg test err: number of txs not match ")
	}

	wantString := "{tx_num: 5}"
	if txsMsg.String() != wantString {
		t.Errorf("txs msg string test err. got:%s want:%s", txsMsg.String(), wantString)
	}

	for i, tx := range txs {
		if !reflect.DeepEqual(tx.Tx, gotTxs[i].Tx) {
			t.Errorf("txs msg test err: got %s\nwant %s", spew.Sdump(tx.Tx), spew.Sdump(gotTxs[i].Tx))
		}
	}
}

var testBlock = &types.Block{
	BlockHeader: types.BlockHeader{
		Version:   1,
		Height:    0,
		Timestamp: 1528945000,
		BlockCommitment: types.BlockCommitment{
			TransactionsMerkleRoot: bc.Hash{V0: uint64(0x11)},
			TransactionStatusHash:  bc.Hash{V0: uint64(0x55)},
		},
	},
}

func TestBlockMessage(t *testing.T) {
	blockMsg, err := NewBlockMessage(testBlock)
	if err != nil {
		t.Fatalf("create new block msg err:%s", err)
	}

	gotBlock, err := blockMsg.GetBlock()
	if err != nil {
		t.Fatalf("got block err:%s", err)
	}

	if !reflect.DeepEqual(gotBlock.BlockHeader, testBlock.BlockHeader) {
		t.Errorf("block msg test err: got %s\nwant %s", spew.Sdump(gotBlock.BlockHeader), spew.Sdump(testBlock.BlockHeader))
	}

	wantString := "{block_height: 0, block_hash: f59514e2541488a38bc2667940bc2c24027e4a3a371d884b55570d036997bb57}"
	if blockMsg.String() != wantString {
		t.Errorf("block msg test err. got:%s want:%s", blockMsg.String(), wantString)
	}

	blockMsg.RawBlock[1] = blockMsg.RawBlock[1] + 0x1
	_, err = blockMsg.GetBlock()
	if err == nil {
		t.Fatalf("get mine block err")
	}

	wantString = "{err: wrong message}"
	if blockMsg.String() != wantString {
		t.Errorf("block msg test err. got:%s want:%s", blockMsg.String(), wantString)
	}
}

var testHeaders = []*types.BlockHeader{
	{
		Version:   1,
		Height:    0,
		Timestamp: 1528945000,
		BlockCommitment: types.BlockCommitment{
			TransactionsMerkleRoot: bc.Hash{V0: uint64(0x11)},
			TransactionStatusHash:  bc.Hash{V0: uint64(0x55)},
		},
	},
	{
		Version:   1,
		Height:    1,
		Timestamp: 1528945000,
		BlockCommitment: types.BlockCommitment{
			TransactionsMerkleRoot: bc.Hash{V0: uint64(0x11)},
			TransactionStatusHash:  bc.Hash{V0: uint64(0x55)},
		},
	},
	{
		Version:   1,
		Height:    3,
		Timestamp: 1528945000,
		BlockCommitment: types.BlockCommitment{
			TransactionsMerkleRoot: bc.Hash{V0: uint64(0x11)},
			TransactionStatusHash:  bc.Hash{V0: uint64(0x55)},
		},
	},
}

func TestHeadersMessage(t *testing.T) {
	headersMsg, err := NewHeadersMessage(testHeaders)
	if err != nil {
		t.Fatalf("create headers msg err:%s", err)
	}

	gotHeaders, err := headersMsg.GetHeaders()
	if err != nil {
		t.Fatalf("got headers err:%s", err)
	}

	if !reflect.DeepEqual(gotHeaders, testHeaders) {
		t.Errorf("headers msg test err: got %s\nwant %s", spew.Sdump(gotHeaders), spew.Sdump(testHeaders))
	}

	wantString := "{header_length: 3}"
	if headersMsg.String() != wantString {
		t.Errorf("headers msg test string err. got:%s want:%s", headersMsg.String(), wantString)
	}

}

func TestGetBlockMessage(t *testing.T) {
	testCase := []struct {
		height     uint64
		rawHash    [32]byte
		wantString string
	}{
		{
			height:     uint64(100),
			rawHash:    [32]byte{0x01},
			wantString: "{height: 100}",
		},
		{
			height:     uint64(0),
			rawHash:    [32]byte{0x01},
			wantString: "{hash: 0100000000000000000000000000000000000000000000000000000000000000}",
		},
	}
	for i, c := range testCase {
		getBlockMsg := NewGetBlockMessage(c.height, c.rawHash)
		gotHash := getBlockMsg.GetHash()

		if !reflect.DeepEqual(gotHash.Byte32(), c.rawHash) {
			t.Errorf("index:%d test get block msg err. got: %s want: %s", i, spew.Sdump(gotHash.Byte32()), spew.Sdump(c.rawHash))
		}

		if getBlockMsg.Height != c.height {
			t.Errorf("index:%d test get block msg err. got: %d want: %d", i, getBlockMsg.Height, c.height)
		}
		if getBlockMsg.String() != c.wantString {
			t.Errorf("index:%d test get block msg string err. got: %s want: %s", i, getBlockMsg.String(), c.wantString)
		}

	}
}

type testGetBlocksMessage struct {
	blockLocator []*bc.Hash
	stopHash     *bc.Hash
}

func TestGetBlocksMessage(t *testing.T) {
	testMsg := testGetBlocksMessage{
		blockLocator: []*bc.Hash{{V0: 0x01}, {V0: 0x02}, {V0: 0x03}},
		stopHash:     &bc.Hash{V0: 0xaa, V2: 0x55},
	}

	getBlocksMsg := NewGetBlocksMessage(testMsg.blockLocator, testMsg.stopHash)
	gotBlockLocator := getBlocksMsg.GetBlockLocator()
	gotStopHash := getBlocksMsg.GetStopHash()

	if !reflect.DeepEqual(gotBlockLocator, testMsg.blockLocator) {
		t.Errorf("get headers msg test err: got %s\nwant %s", spew.Sdump(gotBlockLocator), spew.Sdump(testMsg.blockLocator))
	}

	if !reflect.DeepEqual(gotStopHash, testMsg.stopHash) {
		t.Errorf("get headers msg test err: got %s\nwant %s", spew.Sdump(gotStopHash), spew.Sdump(testMsg.stopHash))
	}

	wantString := "{stop_hash: 00000000000000aa000000000000000000000000000000550000000000000000}"
	if getBlocksMsg.String() != wantString {
		t.Errorf("get headers msg string test err: got:%s want:%s", getBlocksMsg.String(), wantString)
	}
}

type testGetHeadersMessage struct {
	blockLocator []*bc.Hash
	stopHash     *bc.Hash
}

func TestGetHeadersMessage(t *testing.T) {
	testMsg := testGetHeadersMessage{
		blockLocator: []*bc.Hash{{V0: 0x01}, {V0: 0x02}, {V0: 0x03}},
		stopHash:     &bc.Hash{V0: 0xaa, V2: 0x55},
	}
	getHeadersMsg := NewGetHeadersMessage(testMsg.blockLocator, testMsg.stopHash)
	gotBlockLocator := getHeadersMsg.GetBlockLocator()
	gotStopHash := getHeadersMsg.GetStopHash()

	if !reflect.DeepEqual(testMsg.blockLocator, gotBlockLocator) {
		t.Errorf("get headers msg test err: got %s\nwant %s", spew.Sdump(gotBlockLocator), spew.Sdump(testMsg.blockLocator))
	}

	if !reflect.DeepEqual(testMsg.stopHash, gotStopHash) {
		t.Errorf("get headers msg test err: got %s\nwant %s", spew.Sdump(gotStopHash), spew.Sdump(testMsg.stopHash))
	}

	wantString := "{stop_hash: 00000000000000aa000000000000000000000000000000550000000000000000}"
	if getHeadersMsg.String() != wantString {
		t.Errorf("get headers msg string test err: got:%s want:%s", getHeadersMsg.String(), wantString)
	}
}

var testBlocks = []*types.Block{
	{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Timestamp: 1528945000,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: bc.Hash{V0: uint64(0x11)},
				TransactionStatusHash:  bc.Hash{V0: uint64(0x55)},
			},
		},
	},
	{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Timestamp: 1528945000,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: bc.Hash{V0: uint64(0x11)},
				TransactionStatusHash:  bc.Hash{V0: uint64(0x55)},
			},
		},
	},
}

func TestBlocksMessage(t *testing.T) {
	blocksMsg, err := NewBlocksMessage(testBlocks)
	if err != nil {
		t.Fatalf("create blocks msg err:%s", err)
	}
	gotBlocks, err := blocksMsg.GetBlocks()
	if err != nil {
		t.Fatalf("get blocks err:%s", err)
	}

	for _, gotBlock := range gotBlocks {
		if !reflect.DeepEqual(gotBlock.BlockHeader, testBlock.BlockHeader) {
			t.Errorf("block msg test err: got %s\nwant %s", spew.Sdump(gotBlock.BlockHeader), spew.Sdump(testBlock.BlockHeader))
		}
	}

	wantString := "{blocks_length: 2}"
	if blocksMsg.String() != wantString {
		t.Errorf("block msg string test err: got:%s want:%s", blocksMsg.String(), wantString)
	}
}

func TestStatusMessage(t *testing.T) {
	statusResponseMsg := NewStatusMessage(&testBlock.BlockHeader)
	gotHash := statusResponseMsg.GetHash()
	if !reflect.DeepEqual(*gotHash, testBlock.Hash()) {
		t.Errorf("status response msg test err: got %s\nwant %s", spew.Sdump(*gotHash), spew.Sdump(testBlock.Hash()))
	}

	wantString := "{height: 0, hash: f59514e2541488a38bc2667940bc2c24027e4a3a371d884b55570d036997bb57}"
	if statusResponseMsg.String() != wantString {
		t.Errorf("status response msg string test err: got:%s want:%s", statusResponseMsg.String(), wantString)
	}
}

func TestMinedBlockMessage(t *testing.T) {
	blockMsg, err := NewMinedBlockMessage(testBlock)
	if err != nil {
		t.Fatalf("create new mine block msg err:%s", err)
	}

	gotBlock, err := blockMsg.GetMineBlock()
	if err != nil {
		t.Fatalf("got block err:%s", err)
	}

	if !reflect.DeepEqual(gotBlock.BlockHeader, testBlock.BlockHeader) {
		t.Errorf("block msg test err: got %s\nwant %s", spew.Sdump(gotBlock.BlockHeader), spew.Sdump(testBlock.BlockHeader))
	}

	wantString := "{block_height: 0, block_hash: f59514e2541488a38bc2667940bc2c24027e4a3a371d884b55570d036997bb57}"
	if blockMsg.String() != wantString {
		t.Errorf("block msg test err. got:%s want:%s", blockMsg.String(), wantString)
	}

	blockMsg.RawBlock[1] = blockMsg.RawBlock[1] + 0x1
	_, err = blockMsg.GetMineBlock()
	if err == nil {
		t.Fatalf("get mine block err")
	}

	wantString = "{err: wrong message}"
	if blockMsg.String() != wantString {
		t.Errorf("block msg test err. got:%s want:%s", blockMsg.String(), wantString)
	}
}

func TestFilterLoadMessage(t *testing.T) {
	filterLoadMsg := &FilterLoadMessage{
		Addresses: [][]byte{{0x01}, {0x01, 0x02}},
	}

	wantString := "{addresses_length: 2}"
	if filterLoadMsg.String() != wantString {
		t.Errorf("filter load msg test err. got:%s want:%s", filterLoadMsg.String(), wantString)
	}
}

func TestFilterAddMessage(t *testing.T) {
	filterAddMessage := &FilterAddMessage{
		Address: []byte{0x01, 0x02, 0x03},
	}

	wantString := "{address: 010203}"
	if filterAddMessage.String() != wantString {
		t.Errorf("filter add msg test err. got:%s want:%s", filterAddMessage.String(), wantString)
	}
}

func TestFilterClearMessage(t *testing.T) {
	filterClearMessage := &FilterClearMessage{}

	wantString := "{}"
	if filterClearMessage.String() != wantString {
		t.Errorf("filter clear msg test err. got:%s want:%s", filterClearMessage.String(), wantString)
	}
}

func TestGetMerkleBlockMessage(t *testing.T) {
	testCase := []struct {
		height     uint64
		rawHash    [32]byte
		wantString string
	}{
		{
			height:     uint64(100),
			rawHash:    [32]byte{0x01},
			wantString: "{height: 100}",
		},
		{
			height:     uint64(0),
			rawHash:    [32]byte{0x01},
			wantString: "{hash: 0100000000000000000000000000000000000000000000000000000000000000}",
		},
	}
	for i, c := range testCase {
		getMerkleBlockMsg := &GetMerkleBlockMessage{
			Height:  c.height,
			RawHash: c.rawHash,
		}
		gotHash := getMerkleBlockMsg.GetHash()

		if !reflect.DeepEqual(gotHash.Byte32(), c.rawHash) {
			t.Errorf("index:%d test get merkle block msg err. got: %s want: %s", i, spew.Sdump(gotHash.Byte32()), spew.Sdump(c.rawHash))
		}

		if getMerkleBlockMsg.Height != c.height {
			t.Errorf("index:%d test get merkle block msg err. got: %d want: %d", i, getMerkleBlockMsg.Height, c.height)
		}
		if getMerkleBlockMsg.String() != c.wantString {
			t.Errorf("index:%d test get merkle block msg string err. got: %s want: %s", i, getMerkleBlockMsg.String(), c.wantString)
		}
	}
}

func TestMerkleBlockMessage(t *testing.T) {
	blockHeader := types.BlockHeader{
		Version:   1,
		Height:    0,
		Timestamp: 1528945000,
		BlockCommitment: types.BlockCommitment{
			TransactionsMerkleRoot: bc.Hash{V0: uint64(0x11)},
			TransactionStatusHash:  bc.Hash{V0: uint64(0x55)},
		},
	}
	txHashes := []*bc.Hash{{V0: 123, V1: 234, V2: 345, V3: 456}}
	txFlags := []uint8{0x1, 0x2}
	relatedTxs := txs
	statusHashes := []*bc.Hash{{V0: 123, V1: 234, V2: 345, V3: 456}}
	relatedStatuses := []*bc.TxVerifyResult{{StatusFail: false}, {StatusFail: true}}
	merkleBlockMsg := NewMerkleBlockMessage()
	merkleBlockMsg.SetRawBlockHeader(blockHeader)
	merkleBlockMsg.SetTxInfo(txHashes, txFlags, relatedTxs)
	merkleBlockMsg.SetStatusInfo(statusHashes, relatedStatuses)
	if !reflect.DeepEqual(merkleBlockMsg.Flags, txFlags) {
		t.Errorf("test get merkle block msg err. got: %s want: %s", merkleBlockMsg.Flags, txFlags)
	}
	wantString := "{}"
	if merkleBlockMsg.String() != wantString {
		t.Errorf("merkle block msg test err. got:%s want:%s", merkleBlockMsg.String(), wantString)
	}
}
