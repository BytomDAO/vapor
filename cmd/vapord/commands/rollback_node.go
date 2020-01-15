package commands

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/vapor/node"
	"github.com/bytom/vapor/util"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback chain to target height!",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		setLogLevel(config.LogLevel)

		var height int64
		var err error

		height, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(util.ErrLocalExe)
		}
		err = node.Rollback(config, height)
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(util.ErrLocalExe)
		}
	},
}

func init() {
	RootCmd.AddCommand(rollbackCmd)
}
