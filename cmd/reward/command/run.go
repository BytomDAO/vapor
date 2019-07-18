package command

import (
	"fmt"
	"path"
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

var coinBasePeriod uint64

var runRewardCmd = &cobra.Command{
	Use:   "reward",
	Short: "Run the reward",
	RunE:  runReward,
}

var RootCmd = &cobra.Command{
	Use:   "reward",
	Short: "distribution of reward.",
}

func init() {

	runRewardCmd.Flags().Uint64Var(&coinBasePeriod, "coin_base_period", 1, "Consensus cycle")
	RootCmd.AddCommand(runRewardCmd)
}

func runReward(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	configFilePath := path.Join("./", "reward.json")
	config := &cfg.Config{}
	if err := cfg.LoadFederationFile(configFilePath, config); err != nil {
		cmn.Exit(cmn.Fmt("Failed to load reward information:[%s]", err.Error()))
	}

	db, err := common.NewMySQLDB(config.MySQLConfig)
	if err != nil {
		log.WithField("err", err).Panic("initialize mysql db error")
	}

	initActiveNetParams(config)

	sync := synchron.NewChainKeeper(db, config)

	go sync.Run()

	quit := make(chan struct{})
	fmt.Println(coinBasePeriod)
	r := reward.NewReward(db, config, coinBasePeriod, quit)
	r.Start()

	select {
	case <-quit:
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
