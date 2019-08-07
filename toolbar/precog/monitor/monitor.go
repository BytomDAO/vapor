package monitor

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	dbm "github.com/vapor/database/leveldb"

	cfg "github.com/vapor/config"
	"github.com/vapor/p2p"
	// conn "github.com/vapor/p2p/connection"
	"github.com/vapor/p2p/signlib"
	"github.com/vapor/toolbar/precog/config"
	"github.com/vapor/toolbar/precog/database/orm"
)

type monitor struct {
	cfg *config.Config
	db  *gorm.DB
}

func NewMonitor(cfg *config.Config, db *gorm.DB) *monitor {
	return &monitor{
		cfg: cfg,
		db:  db,
	}
}

func (m *monitor) Run() {
	m.updateBootstrapNodes()
	go m.discovery()
	ticker := time.NewTicker(time.Duration(m.cfg.CheckFreqSeconds) * time.Second)
	for ; true; <-ticker.C {
		// TODO: lock?
		m.monitorRountine()
	}
}

// create or update: https://github.com/jinzhu/gorm/issues/1307
func (m *monitor) updateBootstrapNodes() {
	for _, node := range m.cfg.Nodes {
		ormNode := &orm.Node{
			PublicKey: node.PublicKey.String(),
			Alias:     node.Alias,
			Host:      node.Host,
			Port:      node.Port,
		}

		if err := m.db.Where(&orm.Node{PublicKey: ormNode.PublicKey}).
			Assign(&orm.Node{
				Alias: node.Alias,
				Host:  node.Host,
				Port:  node.Port,
			}).FirstOrCreate(ormNode).Error; err != nil {
			log.Error(err)
			continue
		}
	}
}

// TODO:
// implement logic first, and then refactor
// /home/gavin/work/go/src/github.com/vapor/
// p2p/test_util.go
// p2p/switch_test.go
func (m *monitor) discovery() {
	mCfg := cfg.DefaultConfig()
	// TODO: fix
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		log.Fatal(err)
	}
	mCfg.DBPath = dirPath
	defer os.RemoveAll(dirPath)

	// TODO: fix
	mCfg.P2P.ListenAddress = "127.0.1.1:0"
	swPrivKey, err := signlib.NewPrivKey()
	if err != nil {
		log.Fatal(err)
	}

	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	// initSwitchFunc
	sw := p2p.MakeSwitch(mCfg, testDB, swPrivKey, initSwitchFunc)
	sw.Start()
	defer sw.Stop()
}

func (m *monitor) monitorRountine() error {
	// TODO: dail nodes, get lantency & best_height
	// TODO: decide check_height("best best_height" - "confirmations")
	// TODO: get blockhash by check_height, get latency
	// TODO: update lantency, active_time and status
	return nil
}
