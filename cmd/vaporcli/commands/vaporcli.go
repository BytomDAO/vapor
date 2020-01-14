package commands

import (
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/bytom/vapor/util"
)

// vaporcli usage template
var usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:
    {{range .Commands}}{{if (and .IsAvailableCommand (.Name | WalletDisable))}}
    {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}

  available with wallet enable:
    {{range .Commands}}{{if (and .IsAvailableCommand (.Name | WalletEnable))}}
    {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// commandError is an error used to signal different error situations in command handling.
type commandError struct {
	s         string
	userError bool
}

func (c commandError) Error() string {
	return c.s
}

func (c commandError) isUserError() bool {
	return c.userError
}

func newUserError(a ...interface{}) commandError {
	return commandError{s: fmt.Sprintln(a...), userError: true}
}

func newSystemError(a ...interface{}) commandError {
	return commandError{s: fmt.Sprintln(a...), userError: false}
}

func newSystemErrorF(format string, a ...interface{}) commandError {
	return commandError{s: fmt.Sprintf(format, a...), userError: false}
}

// Catch some of the obvious user errors from Cobra.
// We don't want to show the usage message for every error.
// The below may be to generic. Time will show.
var userErrorRegexp = regexp.MustCompile("argument|flag|shorthand")

func isUserError(err error) bool {
	if cErr, ok := err.(commandError); ok && cErr.isUserError() {
		return true
	}

	return userErrorRegexp.MatchString(err.Error())
}

// VaporCmd is Vaporcli's root command.
// Every other command attached to VaporcliCmd is a child command to it.
var VaporcliCmd = &cobra.Command{
	Use:   "vaporcli",
	Short: "Vaporcli is a commond line client for vapor (a.k.a. vapord)",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.SetUsageTemplate(usageTemplate)
			cmd.Usage()
		}
	},
}

// Execute adds all child commands to the root command VaporcliCmd and sets flags appropriately.
func Execute() {

	AddCommands()
	AddTemplateFunc()

	if _, err := VaporcliCmd.ExecuteC(); err != nil {
		os.Exit(util.ErrLocalExe)
	}
}

// AddCommands adds child commands to the root command VaporcliCmd.
func AddCommands() {
	VaporcliCmd.AddCommand(createAccessTokenCmd)
	VaporcliCmd.AddCommand(listAccessTokenCmd)
	VaporcliCmd.AddCommand(deleteAccessTokenCmd)
	VaporcliCmd.AddCommand(checkAccessTokenCmd)

	VaporcliCmd.AddCommand(createAccountCmd)
	VaporcliCmd.AddCommand(deleteAccountCmd)
	VaporcliCmd.AddCommand(listAccountsCmd)
	VaporcliCmd.AddCommand(updateAccountAliasCmd)
	VaporcliCmd.AddCommand(createAccountReceiverCmd)
	VaporcliCmd.AddCommand(listAddressesCmd)
	VaporcliCmd.AddCommand(validateAddressCmd)
	VaporcliCmd.AddCommand(listPubKeysCmd)

	VaporcliCmd.AddCommand(createAssetCmd)
	VaporcliCmd.AddCommand(getAssetCmd)
	VaporcliCmd.AddCommand(listAssetsCmd)
	VaporcliCmd.AddCommand(updateAssetAliasCmd)

	VaporcliCmd.AddCommand(getTransactionCmd)
	VaporcliCmd.AddCommand(listTransactionsCmd)

	VaporcliCmd.AddCommand(getUnconfirmedTransactionCmd)
	VaporcliCmd.AddCommand(listUnconfirmedTransactionsCmd)
	VaporcliCmd.AddCommand(decodeRawTransactionCmd)

	VaporcliCmd.AddCommand(listUnspentOutputsCmd)
	VaporcliCmd.AddCommand(listBalancesCmd)

	VaporcliCmd.AddCommand(rescanWalletCmd)
	VaporcliCmd.AddCommand(walletInfoCmd)

	VaporcliCmd.AddCommand(buildTransactionCmd)
	VaporcliCmd.AddCommand(signTransactionCmd)
	VaporcliCmd.AddCommand(submitTransactionCmd)
	VaporcliCmd.AddCommand(estimateTransactionGasCmd)

	VaporcliCmd.AddCommand(getBlockCountCmd)
	VaporcliCmd.AddCommand(getBlockHashCmd)
	VaporcliCmd.AddCommand(getBlockCmd)
	VaporcliCmd.AddCommand(getBlockHeaderCmd)
	VaporcliCmd.AddCommand(getDifficultyCmd)
	VaporcliCmd.AddCommand(getHashRateCmd)

	VaporcliCmd.AddCommand(createKeyCmd)
	VaporcliCmd.AddCommand(deleteKeyCmd)
	VaporcliCmd.AddCommand(listKeysCmd)
	VaporcliCmd.AddCommand(updateKeyAliasCmd)
	VaporcliCmd.AddCommand(resetKeyPwdCmd)
	VaporcliCmd.AddCommand(checkKeyPwdCmd)

	VaporcliCmd.AddCommand(signMsgCmd)
	VaporcliCmd.AddCommand(verifyMsgCmd)
	VaporcliCmd.AddCommand(decodeProgCmd)

	VaporcliCmd.AddCommand(createTransactionFeedCmd)
	VaporcliCmd.AddCommand(listTransactionFeedsCmd)
	VaporcliCmd.AddCommand(deleteTransactionFeedCmd)
	VaporcliCmd.AddCommand(getTransactionFeedCmd)
	VaporcliCmd.AddCommand(updateTransactionFeedCmd)

	VaporcliCmd.AddCommand(isMiningCmd)
	VaporcliCmd.AddCommand(setMiningCmd)

	VaporcliCmd.AddCommand(netInfoCmd)
	VaporcliCmd.AddCommand(gasRateCmd)

	VaporcliCmd.AddCommand(versionCmd)
}

// AddTemplateFunc adds usage template to the root command VaporcliCmd.
func AddTemplateFunc() {
	walletEnableCmd := []string{
		createAccountCmd.Name(),
		listAccountsCmd.Name(),
		deleteAccountCmd.Name(),
		updateAccountAliasCmd.Name(),
		createAccountReceiverCmd.Name(),
		listAddressesCmd.Name(),
		validateAddressCmd.Name(),
		listPubKeysCmd.Name(),

		createAssetCmd.Name(),
		getAssetCmd.Name(),
		listAssetsCmd.Name(),
		updateAssetAliasCmd.Name(),

		createKeyCmd.Name(),
		deleteKeyCmd.Name(),
		listKeysCmd.Name(),
		resetKeyPwdCmd.Name(),
		checkKeyPwdCmd.Name(),
		signMsgCmd.Name(),

		buildTransactionCmd.Name(),
		signTransactionCmd.Name(),

		getTransactionCmd.Name(),
		listTransactionsCmd.Name(),
		listUnspentOutputsCmd.Name(),
		listBalancesCmd.Name(),

		rescanWalletCmd.Name(),
		walletInfoCmd.Name(),
	}

	cobra.AddTemplateFunc("WalletEnable", func(cmdName string) bool {
		for _, name := range walletEnableCmd {
			if name == cmdName {
				return true
			}
		}
		return false
	})

	cobra.AddTemplateFunc("WalletDisable", func(cmdName string) bool {
		for _, name := range walletEnableCmd {
			if name == cmdName {
				return false
			}
		}
		return true
	})
}
