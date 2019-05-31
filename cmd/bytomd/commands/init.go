package commands

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	cfg "github.com/vapor/config"
	"github.com/vapor/crypto/ed25519/chainkd"
)

var initFilesCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize blockchain",
	Run:   initFiles,
}

func init() {
	initFilesCmd.Flags().String("chain_id", config.ChainID, "Select [vapor] or [solonet]")

	RootCmd.AddCommand(initFilesCmd)
}

func initFiles(cmd *cobra.Command, args []string) {
	configFilePath := path.Join(config.RootDir, "config.toml")
	if _, err := os.Stat(configFilePath); !os.IsNotExist(err) {
		log.WithFields(log.Fields{"module": logModule, "config": configFilePath}).Panic("Already exists config file.")
	}

	switch config.ChainID {
	case "vapor":
		cfg.EnsureRoot(config.RootDir, config.ChainID)
	default:
		cfg.EnsureRoot(config.RootDir, "solonet")
	}

	fedFile := config.FederationFile()
	if _, err := os.Stat(fedFile); !os.IsNotExist(err) {
		log.WithFields(log.Fields{"module": logModule, "config": fedFile}).Panic("Already exists federation file.")
	}

	if err := cfg.ExportFederationFile(fedFile, config); err != nil {
		log.WithFields(log.Fields{"module": logModule, "config": fedFile, "error": err}).Panic("exportFederationFile failed.")
	}

	keyFilePath := path.Join(config.RootDir, config.PrivateKeyFile)
	if _, err := os.Stat(keyFilePath); os.IsNotExist(err) {
		xprv, err := chainkd.NewXPrv(nil)
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Panic("fail on generate private key")
		}

		xprvStr := hex.EncodeToString(xprv[:])
		if err := ioutil.WriteFile(keyFilePath, []byte(xprvStr), 0600); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Panic("fail on save private key")
		}
	}

	log.WithFields(log.Fields{"module": logModule, "config": configFilePath}).Info("Initialized bytom")
}
