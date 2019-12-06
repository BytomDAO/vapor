package main

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/tendermint/tmlibs/cli"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/toolbar/consensusreward"
	cfg "github.com/bytom/vapor/toolbar/consensusreward/config"
)

const logModule = "consensusereward"

var (
	startHeight uint64
	endHeight   uint64
	configFile  string
)

var RootCmd = &cobra.Command{
	Use:   "consensusreward",
	Short: "distribution of reward.",
	RunE:  runReward,
}

func init() {
	RootCmd.Flags().Uint64Var(&startHeight, "start_height", 0, "The starting height of the distributive income reward interval, It is a multiple of the dpos consensus cycle(1200). example: 1200")
	RootCmd.Flags().Uint64Var(&endHeight, "end_height", 0, "The end height of the distributive income reward interval, It is a multiple of the dpos consensus cycle(1200). example: 2400")
	RootCmd.Flags().StringVar(&configFile, "config_file", "reward.json", "config file. default: reward.json")
}

func runReward(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	config := &cfg.Config{}
	if err := cfg.LoadConfigFile(configFile, config); err != nil {
		log.WithFields(log.Fields{"module": logModule, "config": configFile, "error": err}).Fatal("Failded to load config file.")
	}
	if startHeight >= endHeight || startHeight%consensus.ActiveNetParams.RoundVoteBlockNums != 0 || endHeight%consensus.ActiveNetParams.RoundVoteBlockNums != 0 {
		log.Fatal("Please check the height range, which must be multiple of the number of block rounds.")
	}

	s := consensusreward.NewStandbyNodeReward(config, startHeight, endHeight)
	if err := s.Settlement(); err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Fatal("Standby node rewards failure.")
	}

	log.WithFields(log.Fields{
		"module":   logModule,
		"duration": time.Since(startTime),
	}).Info("Standby node reward complete")

	return nil
}

func main() {
	cmd := cli.PrepareBaseCmd(RootCmd, "REWARD", "./")
	cmd.Execute()
}
