package commands

import (
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/bytom/vapor/node"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback chain to target height!",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		setLogLevel(config.LogLevel)

		height, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Fatal("failed to parse int")
		}

		if height < 0 {
			log.WithFields(log.Fields{"module": logModule}).Fatal("height should >= 0")
		}

		if err = node.Rollback(config, uint64(height)); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Fatal("failed to rollback")
		}

		log.WithFields(log.Fields{"module": logModule}).Infof("success to rollback height of %d", height)
	},
}

func init() {
	RootCmd.AddCommand(rollbackCmd)
}
