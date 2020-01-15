package commands

import (
	"fmt"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/bytom/vapor/node"
	"github.com/bytom/vapor/util"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback chain to target height!",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		setLogLevel(config.LogLevel)

		height, err := strconv.ParseInt(args[0], 10, 64)

		fmt.Print("now go to rollback height:", height, "\n")
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Fatal("failed to parse int")
			os.Exit(util.ErrLocalExe)
			fmt.Print("failed to parse int\n")
		}

		if err = node.Rollback(config, height); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Fatal("failed to rollback")
			os.Exit(util.ErrLocalExe)
			fmt.Print("failed to rollback\n")
		}
	},
}

func init() {
	RootCmd.AddCommand(rollbackCmd)
}
