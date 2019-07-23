package main

import (
	"github.com/tendermint/tmlibs/cli"

	"github.com/vapor/cmd/reward/command"
)

func main() {
	cmd := cli.PrepareBaseCmd(command.RootCmd, "REWARD", "./")
	cmd.Execute()
}
