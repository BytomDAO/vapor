package p2p

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/tendermint/go-wire"

	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/p2p/signlib"
)

func mockCompatibleWithFalse(remoteVerStr string) (bool, error) {
	return false, nil
}

func mockCompatibleWithTrue(remoteVerStr string) (bool, error) {
	return true, nil
}

func TestCompatibleWith(t *testing.T) {
	nodeInfo := &NodeInfo{Network: "testnet", NetworkID: 0x888}

	cases := []struct {
		other                 *NodeInfo
		versionCompatibleWith VersionCompatibleWith
		err                   error
	}{
		{other: &NodeInfo{Network: "mainnet", NetworkID: 0x888}, versionCompatibleWith: mockCompatibleWithTrue, err: errDiffNetwork},
		{other: &NodeInfo{Network: "testnet", NetworkID: 0x888}, versionCompatibleWith: mockCompatibleWithTrue, err: nil},
		{other: &NodeInfo{Network: "testnet", NetworkID: 0x999}, versionCompatibleWith: mockCompatibleWithTrue, err: errDiffNetworkID},
		{other: &NodeInfo{Network: "testnet", NetworkID: 0x888}, versionCompatibleWith: mockCompatibleWithFalse, err: errDiffMajorVersion},
	}

	for i, c := range cases {
		if err := nodeInfo.compatibleWith(c.other, c.versionCompatibleWith); errors.Root(err) != c.err {
			t.Fatalf("index:%d node info compatible test err want:%s result:%s", i, c.err, err)
		}
	}
}

func TestNodeInfoWriteRead(t *testing.T) {
	key := [64]byte{0x01, 0x02}
	pubKey, err := signlib.NewPubKey(key[:])
	if err != nil {
		t.Fatal("create pubkey err.", err)
	}
	nodeInfo := &NodeInfo{PubKey: pubKey.String(), Moniker: "vapord", Network: "mainnet", NetworkID: 0x888, RemoteAddr: "127.0.0.2:0", ListenAddr: "127.0.0.1:0", Version: "1.1.0-test", ServiceFlag: 10, Other: []string{"abc", "bcd"}}
	n, err1, err2 := new(int), new(error), new(error)
	buf := new(bytes.Buffer)

	wire.WriteBinary(nodeInfo, buf, n, err1)
	if *err1 != nil {
		t.Fatal(*err1)
	}

	peerNodeInfo := new(NodeInfo)
	wire.ReadBinary(peerNodeInfo, buf, maxNodeInfoSize, new(int), err2)
	if *err2 != nil {
		t.Fatal(*err1)
	}

	if !reflect.DeepEqual(*nodeInfo, *peerNodeInfo) {
		t.Fatal("TestNodeInfoWriteRead err", spew.Sdump(nodeInfo), spew.Sdump(peerNodeInfo))
	}
}
