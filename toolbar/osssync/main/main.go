package main

import (
	"fmt"

	"github.com/bytom/vapor/toolbar/osssync/upload"
)

func main() {
	//NewJsonInfoTxt()
	//return

	//2958741

	if err := upload.Run(); err != nil {
		fmt.Println(err)
	}
}
