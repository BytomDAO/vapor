package types

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/vapor/encoding/blockchain"
	"github.com/vapor/testutil"
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
