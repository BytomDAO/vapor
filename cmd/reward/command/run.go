package command

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/vapor/consensus"
	"github.com/vapor/toolbar/common"
	cfg "github.com/vapor/toolbar/vote_reward/config"
	"github.com/vapor/toolbar/vote_reward/synchron"
)

const logModule = "reward"

var (
	rewardStartHeight uint64
	rewardEndHeight   uint64
	chainID           string
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
	runRewardCmd.Flags().StringVar(&chainID, "chain_id", "mainnet", "")

	RootCmd.AddCommand(runRewardCmd)
}

func runReward(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	configFilePath := cfg.ConfigFile()
	config := &cfg.Config{}
	if err := cfg.LoadConfigFile(configFilePath, config); err != nil {
		log.WithFields(log.Fields{"module": logModule, "config": configFilePath, "error": err}).Fatal("Failded to load config file.")
	}

	initActiveNetParams(config)
	if rewardStartHeight >= rewardEndHeight || rewardStartHeight%consensus.ActiveNetParams.RoundVoteBlockNums != 0 || rewardEndHeight%consensus.ActiveNetParams.RoundVoteBlockNums != 0 {
		log.Fatal("Please check the height range, which must be multiple of the number of block rounds.")
	}

	db, err := common.NewMySQLDB(config.MySQLConfig)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Fatal("Failded to initialize mysql db.")
	}

	sync, err := synchron.NewChainKeeper(db, config, rewardEndHeight)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Fatal("Failded to initialize NewChainKeeper.")
	}

	if err := sync.SyncBlock(); err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Fatal("Failded to sync block.")
	}

	log.WithFields(log.Fields{
		"module":   logModule,
		"duration": time.Since(startTime),
	}).Info("reward complete")

	return nil
}

func initActiveNetParams(config *cfg.Config) {
	var exist bool
	consensus.ActiveNetParams, exist = consensus.NetParams[chainID]
	if !exist {
		cmn.Exit(cmn.Fmt("chain_id[%v] don't exist", chainID))
	}
}
