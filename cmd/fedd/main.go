package main

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database"
	// "github.com/blockcenter/synchron"
)

func main() {
	cfg := config.NewConfig()
	_, err := database.NewMySQLDB(cfg.MySQLConfig)
	if err != nil {
		log.WithField("err", err).Panic("initialize mysql db error")
	}

	// keep the main func running in case of terminating goroutines
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
