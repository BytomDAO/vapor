package main

import (
	"runtime"

	cmd "github.com/vapor/cmd/vaporcli/commands"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	cmd.Execute()
}
