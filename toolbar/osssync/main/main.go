package main

import (
	"fmt"

	"github.com/bytom/vapor/toolbar/osssync/upload"
)

func main() {
	err := upload.Run()
	if err != nil {
		fmt.Println(err)
	}
}
