package main

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/federation"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database"
	"github.com/vapor/federation/database/orm"
	"github.com/vapor/federation/synchron"
)

func main() {
	cfg := config.NewConfig()
	db, err := database.NewMySQLDB(cfg.MySQLConfig)
	if err != nil {
		log.WithField("err", err).Panic("initialize mysql db error")
	}

	txCh := make(chan *orm.CrossTransaction)
	go synchron.NewMainchainKeeper(db, &cfg.Mainchain, txCh).Run()
	go synchron.NewSidechainKeeper(db, &cfg.Sidechain, txCh).Run()
	go federation.NewWarder(cfg, db, txCh).Run()

	// keep the main func running in case of terminating goroutines
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
