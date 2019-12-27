package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/tendermint/tmlibs/cli"

	"github.com/bytom/vapor/toolbar/mergeutxo"
)

var RootCmd = &cobra.Command{
	Use:   "utxomerge",
	Short: "merge utxo.",
	RunE:  runReward,
}

var (
	hostPort, accountID, password, address string
	amount                                 uint64
)

func init() {
	RootCmd.Flags().StringVar(&hostPort, "host_port", "http://127.0.0.1:9889", "The url for the node. Default:http://127.0.0.1:9889")
	RootCmd.Flags().StringVar(&accountID, "account_id", "", "The accountID of utxo needs to be merged")
	RootCmd.Flags().StringVar(&password, "password", "", "Password of the account")
	RootCmd.Flags().StringVar(&address, "address", "", "The received address after merging utxo")
	RootCmd.Flags().Uint64Var(&amount, "amount", 0, "Total amount of merged utxo")
}

func runReward(cmd *cobra.Command, args []string) error {
	log.Info("This tool belongs to an open-source project, we can not guarantee this tool is bug-free. Please check the code before using, developers will not be responsible for any asset loss due to bug!")
	txIDs, err := mergeutxo.MergeUTXO(hostPort, accountID, password, address, amount)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Merge utxo successfully. txID: ", txIDs)

	return nil
}

func main() {
	cmd := cli.PrepareBaseCmd(RootCmd, "merge_utxo", "./")
	cmd.Execute()
}
