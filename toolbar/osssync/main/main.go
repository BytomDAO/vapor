package main

import (
	"fmt"

	"github.com/bytom/vapor/toolbar/osssync/upload"
)

func main() {

	NewJsonInfoTxt()


	return
	err := upload.Run()
	if err != nil {
		fmt.Println(err)
	}
}
