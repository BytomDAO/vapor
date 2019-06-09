package main

import (
	// "encoding/json"
	// "encoding/hex"
	"fmt"
	"reflect"

	// btmTypes "github.com/bytom/protocol/bc/types"

	"github.com/vapor/federation/service"
	// vaporTypes "github.com/vapor/protocol/bc/types"
)

func main() {

	node := service.NewNode("http://127.0.0.1:9888")
	_, b, _ := node.GetBlockByHeight(1)
	// fmt.Println(reflect.TypeOf(a))
	fmt.Println(reflect.TypeOf(b))
	fmt.Println(b)
	// fmt.Println(string(b))
	// fmt.Println(reflect.TypeOf(c))

	// block := &btmTypes.Block{}
	// block.UnmarshalText([]byte(a))
	// fmt.Println(block)

	// block2 := &vaporTypes.Block{}
	// block2.UnmarshalText([]byte(a))
	// fmt.Println(block2)

}
