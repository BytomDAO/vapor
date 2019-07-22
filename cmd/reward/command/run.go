package command

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/vapor/consensus"
	"github.com/vapor/toolbar/common"
	"github.com/vapor/toolbar/reward"
	cfg "github.com/vapor/toolbar/reward/config"
	"github.com/vapor/toolbar/reward/synchron"
)

const logModule = "reward"

var (
	rewardStartHeight uint64
	rewardEndHeight   uint64
)

var (
	RootCmd = &cobra.Command{
		Use:   "reward",
		Short: "distribution of reward.",
	}

	runRewardCmd = &cobra.Command{
		Use:   "reward",
		Short: "Run the reward",
		RunE:  runReward,
	}
)

func init() {
	runRewardCmd.Flags().Uint64Var(&rewardStartHeight, "reward_start_height", 1200, "The starting height of the distributive income reward interval")
	runRewardCmd.Flags().Uint64Var(&rewardEndHeight, "reward_end_height", 2400, "The end height of the distributive income reward interval")

	RootCmd.AddCommand(runRewardCmd)
}

func runReward(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	configFilePath := cfg.ConfigFile()
	config := &cfg.Config{}
	if err := cfg.LoadFederationFile(configFilePath, config); err != nil {
		cmn.Exit(cmn.Fmt("Failed to load reward information:[%s]", err.Error()))
	}

	initActiveNetParams(config)
	if rewardStartHeight >= rewardEndHeight || rewardStartHeight%consensus.ActiveNetParams.RoundVoteBlockNums != 0 || rewardEndHeight%consensus.ActiveNetParams.RoundVoteBlockNums != 0 {
		cmn.Exit("Please check the height range, which must be multiple of the number of block rounds")
	}

	db, err := common.NewMySQLDB(config.MySQLConfig)
	if err != nil {
		cmn.Exit(cmn.Fmt("initialize mysql db error:[%s]", err.Error()))
	}

	sync, err := synchron.NewChainKeeper(db, config, rewardEndHeight)
	if err != nil {
		cmn.Exit(cmn.Fmt("initialize NewChainKeeper error:[%s]", err.Error()))
	}

	if err := sync.Start(); err != nil {
		cmn.Exit(cmn.Fmt("Failded to sync block:[%s]", err.Error()))
	}

	r := reward.NewReward(db, config, rewardStartHeight, rewardEndHeight)
	if err := r.Start(); err != nil {
		cmn.Exit(cmn.Fmt("Failded to send reward:[%s]", err.Error()))
	}

	log.WithFields(log.Fields{
		"module":   logModule,
		"duration": time.Since(startTime),
	}).Info("reward complete")

	return nil
}

func initActiveNetParams(config *cfg.Config) {
	var exist bool
	consensus.ActiveNetParams, exist = consensus.NetParams[config.Chain.ChainID]
	if !exist {
		cmn.Exit(cmn.Fmt("chain_id[%v] don't exist", config.Chain.ChainID))
	}
}
