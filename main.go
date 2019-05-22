package main

import (
	"fmt"
	"github.com/vapor/consensus"
)

func main() {
	for _, x := range consensus.Federation().XPubs {
		fmt.Printf("%T\n%v\n%s\n\n", x, x, x)
	}
}
