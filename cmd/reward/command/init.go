package command

import (
	"os"
	"path"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	cfg "github.com/vapor/toolbar/reward/config"
)

var isVoterReward bool

var initFilesCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize reward",
	Run:   initFiles,
}

func init() {
	initFilesCmd.Flags().BoolVarP(&isVoterReward, "is_voter_reward", "", false, "Is not voting user revenue distribution")

	RootCmd.AddCommand(initFilesCmd)
}

func initFiles(cmd *cobra.Command, args []string) {
	//generate the reward config file
	config := cfg.DefaultConfig(isVoterReward)
	configFilePath := path.Join("./", "reward.json")
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		if err := cfg.ExportFederationFile(configFilePath, config); err != nil {
			log.WithFields(log.Fields{"module": logModule, "config": configFilePath, "error": err}).Fatal("fail on export reward file")
		}
	} else {
		log.WithFields(log.Fields{"module": logModule, "config": configFilePath}).Fatal("Already exists config file.")
	}

	log.WithFields(log.Fields{"module": logModule, "config": configFilePath}).Info("Initialized reward")
}
