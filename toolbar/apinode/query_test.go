//+build apinode

package apinode

import (
	"fmt"
	"testing"
)

func TestNodeListAddresses(t *testing.T) {
	res, err := n.ListAddresses("test10", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
}

func TestNodeListBalances(t *testing.T) {
	res, err := n.ListBalances("test10")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
}

func TestNodeListUtxos(t *testing.T) {
	res, err := n.ListUtxos("test10", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
}

func TestNodeWalletInfo(t *testing.T) {
	res, err := n.WalletInfo()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
}

func TestNodeNetInfo(t *testing.T) {
	res, err := n.NetInfo()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
}

func TestNodeListPeers(t *testing.T) {
	res, err := n.ListPeers()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
}
