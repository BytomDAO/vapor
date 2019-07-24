package main

import (
	"github.com/tendermint/tmlibs/cli"
)

func main() {
	cmd := cli.PrepareBaseCmd(RootCmd, "REWARD", "./")
	cmd.Execute()
}
