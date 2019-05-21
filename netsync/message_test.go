package netsync

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

var testBlock = &types.Block{
	BlockHeader: types.BlockHeader{
		Version:   1,
		Height:    0,
		Timestamp: 1528945000000,
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

	blockMsg.RawBlock[1] = blockMsg.RawBlock[1] + 0x1
	_, err = blockMsg.GetBlock()
	if err == nil {
		t.Fatalf("get mine block err")
	}
}

var testHeaders = []*types.BlockHeader{
	{
		Version:   1,
		Height:    0,
		Timestamp: 1528945000000,
		BlockCommitment: types.BlockCommitment{
			TransactionsMerkleRoot: bc.Hash{V0: uint64(0x11)},
			TransactionStatusHash:  bc.Hash{V0: uint64(0x55)},
		},
	},
	{
		Version:   1,
		Height:    1,
		Timestamp: 1528945000000,
		BlockCommitment: types.BlockCommitment{
			TransactionsMerkleRoot: bc.Hash{V0: uint64(0x11)},
			TransactionStatusHash:  bc.Hash{V0: uint64(0x55)},
		},
	},
	{
		Version:   1,
		Height:    3,
		Timestamp: 1528945000000,
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
}

func TestGetBlockMessage(t *testing.T) {
	getBlockMsg := GetBlockMessage{RawHash: [32]byte{0x01}}
	gotHash := getBlockMsg.GetHash()

	if !reflect.DeepEqual(gotHash.Byte32(), getBlockMsg.RawHash) {
		t.Errorf("get block msg test err: got %s\nwant %s", spew.Sdump(gotHash.Byte32()), spew.Sdump(getBlockMsg.RawHash))
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
}

var testBlocks = []*types.Block{
	{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Timestamp: 1528945000000,
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
			Timestamp: 1528945000000,
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
}

func TestStatusMessage(t *testing.T) {
	statusResponseMsg := NewStatusMessage(&testBlock.BlockHeader)
	gotHash := statusResponseMsg.GetHash()
	if !reflect.DeepEqual(*gotHash, testBlock.Hash()) {
		t.Errorf("status response msg test err: got %s\nwant %s", spew.Sdump(*gotHash), spew.Sdump(testBlock.Hash()))
	}
}
