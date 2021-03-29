package main

import (
	"fmt"

	"github.com/bytom/vapor/toolbar/osssync/upload"
)

func main() {
	if err := upload.Run(); err != nil {
		fmt.Println(err)
	}
}
