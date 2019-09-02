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

func TestNode_CreateKey(t *testing.T) {
	res, err := n.CreateKey("test10", "123456")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(res)
}

func TestNode_CreateAccount(t *testing.T) {
	res, err := n.CreateAccount("test10")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(res)
}

func TestNode_CreateAccountReceiver(t *testing.T) {
	res, err := n.CreateAccountReceiver("test10")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(res)
}
