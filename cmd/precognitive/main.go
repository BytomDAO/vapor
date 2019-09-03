package main

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/toolbar/common"
	"github.com/vapor/toolbar/precognitive/api"
	"github.com/vapor/toolbar/precognitive/config"
	"github.com/vapor/toolbar/precognitive/monitor"
)

func main() {
	cfg := config.NewConfig()
	db, err := common.NewMySQLDB(cfg.MySQLConfig)
	if err != nil {
		log.WithField("err", err).Panic("initialize mysql db error")
	}

	go monitor.NewMonitor(cfg, db).Run()
	go api.NewApiServer(cfg, db).Run()

	// keep the main func running in case of terminating goroutines
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
