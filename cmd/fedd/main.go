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

	assetStore := database.NewAssetStore(db)
	go synchron.NewMainchainKeeper(db, assetStore, cfg).Run()
	go synchron.NewSidechainKeeper(db, assetStore, cfg).Run()
	go federation.NewWarder(db, assetStore, cfg).Run()

	// keep the main func running in case of terminating goroutines
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
