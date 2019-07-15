package main

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/toolbar/federation/api"
	"github.com/vapor/toolbar/federation/config"
	"github.com/vapor/toolbar/federation/database"
	"github.com/vapor/toolbar/common"
	"github.com/vapor/toolbar/federation/synchron"
)

func main() {
	cfg := config.NewConfig()
	db, err := common.NewMySQLDB(cfg.MySQLConfig)
	if err != nil {
		log.WithField("err", err).Panic("initialize mysql db error")
	}

	assetStore := database.NewAssetStore(db)
	go synchron.NewMainchainKeeper(db, assetStore, cfg).Run()
	go synchron.NewSidechainKeeper(db, assetStore, cfg).Run()
	go api.NewServer(db, cfg).Run()

	// keep the main func running in case of terminating goroutines
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
