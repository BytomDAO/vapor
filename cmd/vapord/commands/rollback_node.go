package commands

import (
	"os"
	"strconv"
	"unicode"

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

		isNumber := true
		for _, ch := range args[0] {
			if !unicode.IsNumber(ch) {
				isNumber = false
			}
		}
		if isNumber {
			height, err = strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				jww.ERROR.Println(err)
				os.Exit(util.ErrLocalExe)
			}
			node.NodeRollback(config, height)

		} else {
			jww.ERROR.Printf("Invalid height value")
			os.Exit(util.ErrLocalExe)
		}
	},
}

func init() {
	RootCmd.AddCommand(rollbackCmd)
}
