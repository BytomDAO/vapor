package command

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	cfg "github.com/vapor/toolbar/reward/config"
)

var isVoteReward bool

var initFilesCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize reward",
	Run:   initFiles,
}

func init() {
	initFilesCmd.Flags().BoolVarP(&isVoteReward, "is_vote_reward", "", false, "Is not voting user revenue distribution")

	RootCmd.AddCommand(initFilesCmd)
}

func initFiles(cmd *cobra.Command, args []string) {
	//generate the reward config file
	config := cfg.DefaultConfig(isVoteReward)
	configFilePath := cfg.ConfigFile()
	if _, err := os.Stat(configFilePath); !os.IsNotExist(err) {
		log.WithFields(log.Fields{"module": logModule, "config": configFilePath}).Fatal("Already exists config file.")
	}

	if err := cfg.ExportConfigFile(configFilePath, config); err != nil {
		log.WithFields(log.Fields{"module": logModule, "config": configFilePath, "error": err}).Fatal("fail on export reward file")
	}

	log.WithFields(log.Fields{"module": logModule, "config": configFilePath}).Info("Initialized reward")
}
