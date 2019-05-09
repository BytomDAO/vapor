package main

import (
	"runtime"

	cmd "github.com/vapor/cmd/bytomcli/commands"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	cmd.Execute()
}
