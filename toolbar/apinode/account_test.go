package apinode

import (
	"fmt"
	"testing"

	"github.com/vapor/crypto/ed25519/chainkd"
)

func TestCreateKey(t *testing.T) {
	n:=NewNode("http://127.0.0.1:9889")
	resp,err:=n.CreateKey("test4","123456","","")
	if err!=nil{
		t.Fatal(err)
	}

	respAccount,err:=n.CreateAccount([]chainkd.XPub{resp.XPub},1,resp.Alias)
	if err!=nil{
		t.Fatal(err)
	}

	fmt.Println(respAccount)
}
