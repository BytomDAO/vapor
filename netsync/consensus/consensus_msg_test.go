package consensus

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/tendermint/go-wire"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

var _ = wire.RegisterInterface(
	struct{ ConsensusMessage }{},
	wire.ConcreteType{&BlockSignMessage{}, BlockSignByte},
	wire.ConcreteType{&BlockProposeMessage{}, BlockProposeByte},
)

func TestDecodeMessage(t *testing.T) {
	testCases := []struct {
		msg     ConsensusMessage
		msgType byte
	}{
		{
			msg: &BlockSignMessage{
				BlockID: [32]byte{0x01},
				Height:  uint64(100),
				Sign:    []byte{0x00},
				Pubkey:  []byte{0x01},
			},
			msgType: BlockSignByte,
		},
		{
			msg: &BlockProposeMessage{
				RawBlock: []byte{0x01, 0x02},
			},
			msgType: BlockProposeByte,
		},
	}
	for i, c := range testCases {
		binMsg := wire.BinaryBytes(struct{ ConsensusMessage }{c.msg})
		gotMsgType, gotMsg, err := decodeMessage(binMsg)
		if err != nil {
			t.Fatalf("index:%d decode Message err %s", i, err)
		}
		if gotMsgType != c.msgType {
			t.Fatalf("index:%d decode Message type err. got:%d want:%d", i, gotMsgType, c.msg)
		}
		if !reflect.DeepEqual(gotMsg, c.msg) {
			t.Fatalf("index:%d decode Message err. got:%s\n want:%s", i, spew.Sdump(gotMsg), spew.Sdump(c.msg))
		}
	}
}

func TestBlockSignBroadcastMsg(t *testing.T) {
	blockSignMsg := &BlockSignMessage{
		BlockID: [32]byte{0x01},
		Height:  uint64(100),
		Sign:    []byte{0x00},
		Pubkey:  []byte{0x01},
	}
	blockSignBroadcastMsg := NewBlockSignBroadcastMsg(blockSignMsg.BlockID, blockSignMsg.Height, blockSignMsg.Sign, blockSignMsg.Pubkey, ConsensusChannel)

	binMsg := wire.BinaryBytes(blockSignBroadcastMsg.GetMsg())
	gotMsgType, gotMsg, err := decodeMessage(binMsg)
	if err != nil {
		t.Fatalf("decode Message err %s", err)
	}
	if gotMsgType != BlockSignByte {
		t.Fatalf("decode Message type err. got:%d want:%d", gotMsgType, BlockSignByte)
	}
	if !reflect.DeepEqual(gotMsg, blockSignMsg) {
		t.Fatalf("decode Message err. got:%s\n want:%s", spew.Sdump(gotMsg), spew.Sdump(blockSignMsg))
	}
}

func TestBlockProposeBroadcastMsg(t *testing.T) {
	blockProposedmsg, _ := NewBlockProposeMessage(testBlock)

	BlockProposeBroadcastMsg, _ := NewBlockProposeBroadcastMsg(testBlock, ConsensusChannel)

	binMsg := wire.BinaryBytes(BlockProposeBroadcastMsg.GetMsg())
	gotMsgType, gotMsg, err := decodeMessage(binMsg)
	if err != nil {
		t.Fatalf("decode Message err %s", err)
	}
	if gotMsgType != BlockProposeByte {
		t.Fatalf("decode Message type err. got:%d want:%d", gotMsgType, BlockProposeByte)
	}
	if !reflect.DeepEqual(gotMsg, blockProposedmsg) {
		t.Fatalf("decode Message err. got:%s\n want:%s", spew.Sdump(gotMsg), spew.Sdump(blockProposedmsg))
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

func TestBlockProposeMessage(t *testing.T) {
	blockMsg, err := NewBlockProposeMessage(testBlock)
	if err != nil {
		t.Fatalf("create new mine block msg err:%s", err)
	}

	gotBlock, err := blockMsg.GetProposeBlock()
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
	_, err = blockMsg.GetProposeBlock()
	if err == nil {
		t.Fatalf("get mine block err")
	}

	wantString = "{err: wrong message}"
	if blockMsg.String() != wantString {
		t.Errorf("block msg test err. got:%s want:%s", blockMsg.String(), wantString)
	}
}

func TestBlockSignMessage(t *testing.T) {
	msg := &BlockSignMessage{
		BlockID: [32]byte{0x01},
		Height:  uint64(100),
		Sign:    []byte{0x00},
		Pubkey:  []byte{0x01},
	}

	gotMsg := NewBlockSignMessage(msg.BlockID, msg.Height, msg.Sign, msg.Pubkey)

	if !reflect.DeepEqual(gotMsg, msg) {
		t.Fatalf("test block sign message err. got:%s\n want:%s", spew.Sdump(gotMsg), spew.Sdump(msg))
	}
	wantString := "{block_hash: 0100000000000000000000000000000000000000000000000000000000000000,block_height: 100,sign:00,pubkey:01}"
	if gotMsg.String() != wantString {
		t.Fatalf("test block sign message err. got string:%s\n want string:%s", gotMsg.String(), wantString)
	}
}
