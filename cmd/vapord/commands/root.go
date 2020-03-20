package commands

import (
	"os/user"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cmn "github.com/tendermint/tmlibs/common"

	cfg "github.com/bytom/vapor/config"
)

var (
	config = cfg.DefaultConfig()
)

// RootCmd is the command for run node
var RootCmd = &cobra.Command{
	Use:   "vapord",
	Short: "Multiple asset management.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		err := viper.Unmarshal(config)
		if err != nil {
			return err
		}
		pathParts := strings.SplitN(config.RootDir, "/", 2)
		if len(pathParts) == 2 && (pathParts[0] == "~" || pathParts[0] == "$HOME") {
			usr, err := user.Current()
			if err != nil {
				cmn.Exit("Error: " + err.Error())
			}
			pathParts[0] = usr.HomeDir
			config.RootDir = strings.Join(pathParts, "/")
		}
		config.SetRoot(config.RootDir)
		return nil
	},
}
