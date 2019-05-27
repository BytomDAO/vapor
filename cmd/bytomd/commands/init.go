package commands

import (
	"encoding/json"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	cfg "github.com/vapor/config"
)

var initFilesCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize blockchain",
	Run:   initFiles,
}

func init() {
	initFilesCmd.Flags().String("chain_id", config.ChainID, "Select [mainnet] or [testnet] or [solonet] or [vapor]")

	RootCmd.AddCommand(initFilesCmd)
}

func initFiles(cmd *cobra.Command, args []string) {
	configFilePath := path.Join(config.RootDir, "config.toml")
	if _, err := os.Stat(configFilePath); !os.IsNotExist(err) {
		log.WithFields(log.Fields{"module": logModule, "config": configFilePath}).Info("Already exists config file.")
		return
	}

	switch config.ChainID {
	case "mainnet", "testnet", "vapor":
		cfg.EnsureRoot(config.RootDir, config.ChainID)
	default:
		cfg.EnsureRoot(config.RootDir, "solonet")
	}

	fedFile := config.FederationFile()
	if _, err := os.Stat(fedFile); !os.IsNotExist(err) {
		log.WithFields(log.Fields{"module": logModule, "config": fedFile}).Info("Already exists federation file.")
		return
	}

	if err := exportFederationFile(fedFile); err != nil {
		log.WithFields(log.Fields{"module": logModule, "config": fedFile, "error": err}).Info("exportFederationFile failed.")
		return
	}

	log.WithFields(log.Fields{"module": logModule, "config": configFilePath}).Info("Initialized bytom")
}

func exportFederationFile(fedFile string) error {
	file, err := os.Open(fedFile)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(config.Federation)
}
