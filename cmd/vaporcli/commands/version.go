package commands

import (
	"runtime"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/bytom/vapor/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of vaporcli",
	Run: func(cmd *cobra.Command, args []string) {
		jww.FEEDBACK.Printf("vaporcli v%s %s/%s\n", version.Version, runtime.GOOS, runtime.GOARCH)
	},
}
