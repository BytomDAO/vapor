package monitor

import (
	// "encoding/binary"
	"io/ioutil"
	"os"
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	// dbm "github.com/vapor/database/leveldb"

	vaporCfg "github.com/vapor/config"
	"github.com/vapor/p2p"
	// conn "github.com/vapor/p2p/connection"
	// "github.com/vapor/consensus"
	// "github.com/vapor/crypto/sha3pool"
	"github.com/vapor/p2p/discover/dht"
	"github.com/vapor/p2p/discover/mdns"
	"github.com/vapor/p2p/signlib"
	"github.com/vapor/toolbar/precog/config"
	"github.com/vapor/toolbar/precog/database/orm"
)

const netID = 0

type monitor struct {
	cfg     *config.Config
	db      *gorm.DB
	nodeCfg *vaporCfg.Config
}

func NewMonitor(cfg *config.Config, db *gorm.DB) *monitor {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		log.Fatal(err)
	}

	nodeCfg := &vaporCfg.Config{
		BaseConfig: vaporCfg.DefaultBaseConfig(),
		P2P:        vaporCfg.DefaultP2PConfig(),
		Federation: vaporCfg.DefaultFederationConfig(),
	}
	nodeCfg.DBPath = dirPath

	return &monitor{
		cfg:     cfg,
		db:      db,
		nodeCfg: nodeCfg,
	}
}

func (m *monitor) Run() {
	defer os.RemoveAll(m.nodeCfg.DBPath)

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
	sw, err := m.makeSwitch()
	if err != nil {
		log.Fatal(err)
	}

	sw.Start()
	defer sw.Stop()
}

func (m *monitor) makeSwitch() (*p2p.Switch, error) {
	// TODO: whatz that for
	// testDB := dbm.NewDB("testdb", "leveldb", dirPath)

	swPrivKey, err := signlib.NewPrivKey()
	if err != nil {
		return nil, err
	}

	l, listenAddr := p2p.GetListener(m.nodeCfg.P2P)
	discv, err := dht.NewDiscover(m.nodeCfg, swPrivKey, l.ExternalAddress().Port, netID)
	if err != nil {
		return nil, err
	}

	lanDiscv := mdns.NewLANDiscover(mdns.NewProtocol(), int(l.ExternalAddress().Port))
	return p2p.NewSwitch(m.nodeCfg, discv, lanDiscv, l, swPrivKey, listenAddr, netID)
}

func (m *monitor) monitorRountine() error {
	// TODO: dail nodes, get lantency & best_height
	// TODO: decide check_height("best best_height" - "confirmations")
	// TODO: get blockhash by check_height, get latency
	// TODO: update lantency, active_time and status
	return nil
}
