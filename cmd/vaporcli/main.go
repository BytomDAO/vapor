package main

import (
	"runtime"

	cmd "github.com/bytom/vapor/cmd/vaporcli/commands"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	cmd.Execute()
}
