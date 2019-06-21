package main

import (
	"fmt"
)

func main() {
	bs := []byte{0}
	inner(bs)
	fmt.Println(bs)
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

// # TODO
// + refactor string
// + request sign
// + maybe witness
// + xprv

// + election
// + withdrawal
