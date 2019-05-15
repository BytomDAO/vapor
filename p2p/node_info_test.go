package p2p

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"

	"github.com/vapor/errors"
)

func mockCompatibleWithFalse(remoteVerStr string) (bool, error) {
	return false, nil
}

func mockCompatibleWithTrue(remoteVerStr string) (bool, error) {
	return true, nil
}

func TestCompatibleWith(t *testing.T) {
	nodeInfo := &NodeInfo{Network: "testnet",NetworkID:0x12345}

	cases := []struct {
		other                 *NodeInfo
		versionCompatibleWith VersionCompatibleWith
		err                   error
	}{
		{other: &NodeInfo{Network: "mainnet",NetworkID:0x12345}, versionCompatibleWith: mockCompatibleWithTrue, err: errDiffNetwork},
		{other: &NodeInfo{Network: "testnet",NetworkID:0x12345}, versionCompatibleWith: mockCompatibleWithTrue, err: nil},
		{other: &NodeInfo{Network: "testnet",NetworkID:0x23456}, versionCompatibleWith: mockCompatibleWithTrue, err: errDiffNetworkID},
		{other: &NodeInfo{Network: "testnet",NetworkID:0x12345}, versionCompatibleWith: mockCompatibleWithFalse, err: errDiffMajorVersion},
	}

	for i, c := range cases {
		if err := nodeInfo.compatibleWith(c.other, c.versionCompatibleWith); errors.Root(err) != c.err {
			t.Fatalf("index %d node info compatible test err want:%s result:%s", i, c.err, errors.Root(err))
		}
	}
}

func TestNodeInfoWriteRead(t *testing.T) {
	nodeInfo := &NodeInfo{PubKey: crypto.GenPrivKeyEd25519().PubKey().Unwrap().(crypto.PubKeyEd25519), Moniker: "bytomd", Network: "mainnet", ListenAddr: "127.0.0.1:0", Version: "1.1.0-test", ServiceFlag: 10, Other: []string{"abc", "bcd"}}
	n, err, err1 := new(int), new(error), new(error)
	buf := new(bytes.Buffer)

	wire.WriteBinary(nodeInfo, buf, n, err)
	if *err != nil {
		t.Fatal(*err)
	}

	peerNodeInfo := new(NodeInfo)
	wire.ReadBinary(peerNodeInfo, buf, maxNodeInfoSize, new(int), err1)
	if *err1 != nil {
		t.Fatal(*err1)
	}

	if !reflect.DeepEqual(*nodeInfo, *peerNodeInfo) {
		t.Fatal("TestNodeInfoWriteRead err")
	}
}
