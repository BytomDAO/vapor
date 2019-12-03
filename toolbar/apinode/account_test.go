//+build apinode

package apinode

import (
	"fmt"
	"testing"
)

var n *Node

func TestMain(m *testing.M) {
	n = NewNode("http://127.0.0.1:9889")
	m.Run()
}

func TestNodeCreateKey(t *testing.T) {
	res, err := n.CreateKey("test10", "123456")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(res)
}

func TestNodeCreateAccount(t *testing.T) {
	res, err := n.CreateAccount("test10", "test11")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(res)
}

func TestNodeCreateAccountReceiver(t *testing.T) {
	res, err := n.CreateAccountReceiver("test10")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(res)
}

func TestNodeListAccounts(t *testing.T) {
	res, err := n.ListAccounts()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(res)
}
