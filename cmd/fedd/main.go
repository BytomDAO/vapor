package main

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database"
	"github.com/vapor/federation/synchron"
)

func main() {
	cfg := config.NewConfig()
	db, err := database.NewMySQLDB(cfg.MySQLConfig)
	if err != nil {
		log.WithField("err", err).Panic("initialize mysql db error")
	}

	go synchron.NewBlockKeeper(db, &cfg.Mainchain).Run()
	go synchron.NewBlockKeeper(db, &cfg.Sidechain).Run()

	// keep the main func running in case of terminating goroutines
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
