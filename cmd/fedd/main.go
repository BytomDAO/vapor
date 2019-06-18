package main

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/federation"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database"
	"github.com/vapor/federation/synchron"
)

// TODO: should we rename bc package
// https://github.com/golang/protobuf/issues/172
func main() {
	cfg := config.NewConfig()
	db, err := database.NewMySQLDB(cfg.MySQLConfig)
	if err != nil {
		log.WithField("err", err).Panic("initialize mysql db error")
	}

	assetKeeper := database.NewAssetKeeper(db)
	go synchron.NewMainchainKeeper(db, assetKeeper, cfg).Run()
	go synchron.NewSidechainKeeper(db, assetKeeper, cfg).Run()
	go federation.NewWarder(db, assetKeeper, cfg).Run()

	// keep the main func running in case of terminating goroutines
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
