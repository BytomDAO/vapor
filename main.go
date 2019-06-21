// TODO: delete this file
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func main() {
	// bs := []byte{0}
	// inner(bs)
	// fmt.Println(bs)

	// var
	raw := [][]byte{[]byte{1, 3}, []byte{2}}
	// // fmt.Println(raw)

	// str := "["
	// str += "]"

	var store []string
	for _, part := range raw {
		store = append(store, hex.EncodeToString(part))
	}

	b, _ /*err*/ := json.Marshal(store)
	fmt.Println(string(b))
	// fmt.Println(b)

	var store2 []string
	json.Unmarshal(b, &store2)
	fmt.Println(store2)

	var raw2 [][]byte
	for _, part := range store2 {
		b, _ := hex.DecodeString(part)
		raw2 = append(raw2, b)
	}

	fmt.Println(raw)
	fmt.Println(raw2)
}

func inner(bs []byte) {
	bs[0] = byte(1)
}

// # DONE
// + listen & build
//     + fix witnesss
//     + sign and submit by hand
//     + pass test
// + refactor asset
// + sign by code
// + refactor string

// # TODO
// + request sign
// + maybe witness
// + xprv

// + election
// + withdrawal
