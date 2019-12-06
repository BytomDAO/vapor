package types

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/bytom/vapor/encoding/blockchain"
	"github.com/bytom/vapor/testutil"
)

func TestReadWriteBlockWitness(t *testing.T) {
	cases := []struct {
		bw        BlockWitness
		hexString string
	}{
		{
			bw:        BlockWitness{Witness: [][]byte{[]byte{0xbe, 0xef}}},
			hexString: "0102beef",
		},
		{
			bw:        BlockWitness{Witness: [][]byte{[]byte{0xbe, 0xef}, []byte{0xab, 0xcd}, []byte{0xcd, 0x68}}},
			hexString: "0302beef02abcd02cd68",
		},
		{
			bw:        BlockWitness{Witness: [][]byte{[]byte{0xbe, 0xef}, nil, []byte{0xcd, 0x68}}},
			hexString: "0302beef0002cd68",
		},
		{
			bw:        BlockWitness{Witness: [][]byte{[]byte{}}},
			hexString: "0100",
		},
		{
			bw:        BlockWitness{},
			hexString: "00",
		},
	}

	for _, c := range cases {
		buff := []byte{}
		buffer := bytes.NewBuffer(buff)
		if err := c.bw.writeTo(buffer); err != nil {
			t.Fatal(err)
		}

		hexString := hex.EncodeToString(buffer.Bytes())
		if hexString != c.hexString {
			t.Errorf("test write block commitment fail, got:%s, want:%s", hexString, c.hexString)
		}

		bc := &BlockWitness{}
		if err := bc.readFrom(blockchain.NewReader(buffer.Bytes())); err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(*bc, c.bw) {
			t.Errorf("test read block commitment fail, got:%v, want:%v", *bc, c.bw)
		}
	}
}

func TestBlockWitnessSet(t *testing.T) {
	cases := []struct {
		bw    BlockWitness
		index uint64
		data  []byte
		want  BlockWitness
	}{
		{
			bw:    BlockWitness{Witness: [][]byte{}},
			index: uint64(0),
			data:  []byte{0x01, 0x02, 0x03, 0x04},
			want:  BlockWitness{Witness: [][]byte{[]byte{0x01, 0x02, 0x03, 0x04}}},
		},
		{
			bw:    BlockWitness{Witness: [][]byte{[]byte{0x01, 0x02, 0x03, 0x04}}},
			index: uint64(1),
			data:  []byte{0x01, 0x01, 0x01, 0x01},
			want:  BlockWitness{Witness: [][]byte{[]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x01, 0x01, 0x01}}},
		},
		{
			bw:    BlockWitness{Witness: [][]byte{[]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}}},
			index: uint64(4),
			data:  []byte{0x04, 0x04, 0x04, 0x04},
			want:  BlockWitness{Witness: [][]byte{[]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}, nil, nil, []byte{0x04, 0x04, 0x04, 0x04}}},
		},
	}

	for i, c := range cases {
		newbw := c.bw
		newbw.Set(c.index, c.data)
		if !testutil.DeepEqual(c.want, newbw) {
			t.Errorf("update result mismatch: %v, got:%v, want:%v", i, newbw, c.want)
		}
	}
}

func TestBlockWitnessDelete(t *testing.T) {
	cases := []struct {
		bw    BlockWitness
		index uint64
		want  BlockWitness
	}{
		{
			bw:    BlockWitness{Witness: [][]byte{}},
			index: uint64(0),
			want:  BlockWitness{Witness: [][]byte{}},
		},
		{
			bw:    BlockWitness{Witness: [][]byte{[]byte{0x01, 0x02, 0x03, 0x04}}},
			index: uint64(0),
			want:  BlockWitness{Witness: [][]byte{[]byte{}}},
		},
		{
			bw:    BlockWitness{Witness: [][]byte{[]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}}},
			index: uint64(1),
			want:  BlockWitness{Witness: [][]byte{[]byte{0x01, 0x02, 0x03, 0x04}, []byte{}}},
		},
		{
			bw:    BlockWitness{Witness: [][]byte{[]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}}},
			index: uint64(100),
			want:  BlockWitness{Witness: [][]byte{[]byte{0x01, 0x02, 0x03, 0x04}, []byte{0x01, 0x02, 0x03, 0x04}}},
		},
	}

	for i, c := range cases {
		newbw := c.bw
		newbw.Delete(c.index)
		if !testutil.DeepEqual(c.want, newbw) {
			t.Errorf("update result mismatch: %v, got:%v, want:%v", i, newbw, c.want)
		}
	}
}
