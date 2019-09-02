//+build apinode

package apinode

import (
	"fmt"
	"testing"
)

func TestNode_ListAddresses(t *testing.T) {
	res, err := n.ListAddresses("test10", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
}

func TestNode_ListBalances(t *testing.T) {
	res, err := n.ListBalances("test10")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
}

func TestNode_ListUtxos(t *testing.T) {
	res, err := n.ListUtxos("test10", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
}

func TestNode_WalletInfo(t *testing.T) {
	res, err := n.WalletInfo()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
}

func TestNode_NetInfo(t *testing.T) {
	res, err := n.NetInfo()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
}

func TestNode_ListPeers(t *testing.T) {
	res, err := n.ListPeers()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
}
