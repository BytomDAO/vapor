package main

import (
	"fmt"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/testutil"
)

func main() {
	fmt.Println((&bc.Hash{V0: 1, V1: 2, V2: 3, V3: 5}).String())
	h := testutil.MustDecodeHash("0000000000000001000000000000000200000000000000030000000000000005")

	fmt.Println(h)
}
