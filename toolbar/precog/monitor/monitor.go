package monitor

import (
	// "encoding/binary"
	// "encoding/hex"
	// "io/ioutil"
	"fmt"
	"net"
	"os"
	// "strings"
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	// dbm "github.com/vapor/database/leveldb"

	vaporCfg "github.com/vapor/config"
	"github.com/vapor/crypto/ed25519/chainkd"
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

var (
	nodesToDiscv = 150
	discvFreqSec = 1
)

type monitor struct {
	cfg     *config.Config
	db      *gorm.DB
	nodeCfg *vaporCfg.Config
	sw      *p2p.Switch
	discvCh chan *dht.Node
	privKey chainkd.XPrv
}

func NewMonitor(cfg *config.Config, db *gorm.DB) *monitor {
	nodeCfg := &vaporCfg.Config{
		BaseConfig: vaporCfg.DefaultBaseConfig(),
		P2P:        vaporCfg.DefaultP2PConfig(),
		Federation: vaporCfg.DefaultFederationConfig(),
	}
	nodeCfg.DBPath = "vapor_precog_data"
	nodeCfg.ChainID = "mainnet"
	discvCh := make(chan *dht.Node)
	privKey, err := signlib.NewPrivKey()
	if err != nil {
		log.Fatal(err)
	}

	return &monitor{
		cfg:     cfg,
		db:      db,
		nodeCfg: nodeCfg,
		discvCh: discvCh,
		privKey: privKey.(chainkd.XPrv),
	}
}

func (m *monitor) Run() {
	defer os.RemoveAll(m.nodeCfg.DBPath)

	for _, node := range m.cfg.Nodes {
		m.upSertNode(&node)
	}

	sw, err := m.makeSwitch()
	if err != nil {
		log.Fatal(err)
	}

	m.sw = sw

	go m.discovery()
	go m.collectDiscv()

	m.cfg.CheckFreqSeconds = 60
	ticker := time.NewTicker(time.Duration(m.cfg.CheckFreqSeconds) * time.Second)
	for ; true; <-ticker.C {
		// TODO: lock?
		m.monitorRountine()
	}
}

// create or update: https://github.com/jinzhu/gorm/issues/1307
func (m *monitor) upSertNode(node *config.Node) error {
	if node.XPub != nil {
		node.PublicKey = fmt.Sprintf("%v", node.XPub.PublicKey().String())
	}

	ormNode := &orm.Node{PublicKey: node.PublicKey}
	if err := m.db.Where(&orm.Node{PublicKey: node.PublicKey}).First(ormNode).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	if node.Alias != "" {
		ormNode.Alias = node.Alias
	}
	if node.XPub != nil {
		ormNode.Xpub = node.XPub.String()
	}
	ormNode.Host = node.Host
	ormNode.Port = node.Port
	return m.db.Where(&orm.Node{PublicKey: ormNode.PublicKey}).
		Assign(&orm.Node{
			Xpub:  ormNode.Xpub,
			Alias: ormNode.Alias,
			Host:  ormNode.Host,
			Port:  ormNode.Port,
		}).FirstOrCreate(ormNode).Error
}

func (m *monitor) makeSwitch() (*p2p.Switch, error) {
	l, listenAddr := p2p.GetListener(m.nodeCfg.P2P)
	discv, err := dht.NewDiscover(m.nodeCfg, m.privKey, l.ExternalAddress().Port, m.cfg.NetworkID)
	if err != nil {
		return nil, err
	}

	// no need for lanDiscv, but passing &mdns.LANDiscover{} will cause NilPointer
	lanDiscv := mdns.NewLANDiscover(mdns.NewProtocol(), int(l.ExternalAddress().Port))
	return p2p.NewSwitch(m.nodeCfg, discv, lanDiscv, l, m.privKey, listenAddr, m.cfg.NetworkID)
}

func (m *monitor) discovery() {
	ticker := time.NewTicker(time.Duration(discvFreqSec) * time.Second)
	for range ticker.C {
		nodes := make([]*dht.Node, nodesToDiscv)
		n := m.sw.GetDiscv().ReadRandomNodes(nodes)
		for i := 0; i < n; i++ {
			m.discvCh <- nodes[i]
		}
	}
}

func (m *monitor) collectDiscv() {
	// nodeMap maps a node's public key to the node itself
	nodeMap := make(map[string]*dht.Node)
	for node := range m.discvCh {
		if n, ok := nodeMap[node.ID.String()]; ok && n.String() == node.String() {
			continue
		}
		log.Info("discover new node: ", node)

		m.upSertNode(&config.Node{
			PublicKey: node.ID.String(),
			Host:      node.IP.String(),
			Port:      node.TCP,
		})
		nodeMap[node.ID.String()] = node
	}
}

func (m *monitor) monitorRountine() error {
	var nodes []*orm.Node
	if err := m.db.Model(&orm.Node{}).Find(&nodes).Error; err != nil {
		return err
	}

	addresses := make([]*p2p.NetAddress, 0)
	for i := 0; i < len(nodes); i++ {
		ip, err := net.LookupIP(nodes[i].Host)
		if err != nil || len(ip) == 0 {
			continue
		}

		address := p2p.NewNetAddressIPPort(ip[0], nodes[i].Port)
		addresses = append(addresses, address)
	}

	m.sw.DialPeers(addresses)

	// TODO: dail nodes, get lantency & best_height
	// TODO: decide check_height("best best_height" - "confirmations")
	// TODO: get blockhash by check_height, get latency
	// TODO: update lantency, active_time and status
	return nil
}

// TODO:
// implement logic first, and then refactor
// /home/gavin/work/go/src/github.com/vapor/
// p2p/test_util.go
// p2p/switch_test.go
// syncManager
// notificationMgr
