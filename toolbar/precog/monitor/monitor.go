package monitor

import (
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	vaporCfg "github.com/vapor/config"
	"github.com/vapor/crypto/ed25519/chainkd"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/event"
	"github.com/vapor/netsync/chainmgr"
	"github.com/vapor/netsync/consensusmgr"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/p2p"
	"github.com/vapor/p2p/discover/dht"
	"github.com/vapor/p2p/discover/mdns"
	"github.com/vapor/p2p/signlib"
	"github.com/vapor/test/mock"
	"github.com/vapor/toolbar/precog/config"
)

// TODO:
// 1. moniker 理论是安全的，只是记得测试一下，这么改不会让vapor node出坑
// 3. toolbar/precog/monitor/stats.go FirstOrCreate&Update 弱弱的问一下，直接save会出事么？
// 4. 碰到一个玄学问题，究竟是以ip为单位，还是pubkey为单位。 如果同一个pubkey出现在2个不同的ip，会不会让数据混乱？
// 6. ***NodeLiveness应该是存每次的通讯记录？至于一些统计数据之类的都丢node上去？
// 7. m这个为什么需要锁呀？一个是节点发现，一个是生命探测，中间交互都是数据库把？

type monitor struct {
	cfg            *config.Config
	db             *gorm.DB
	nodeCfg        *vaporCfg.Config
	sw             *p2p.Switch
	privKey        chainkd.XPrv
	chain          *mock.Chain
	txPool         *mock.Mempool
	bestHeightSeen uint64
	peers          *peers.PeerSet
}

func NewMonitor(cfg *config.Config, db *gorm.DB) *monitor {
	dbPath, err := makePath()
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("makePath")
	}

	nodeCfg := &vaporCfg.Config{
		BaseConfig: vaporCfg.DefaultBaseConfig(),
		P2P:        vaporCfg.DefaultP2PConfig(),
		Federation: vaporCfg.DefaultFederationConfig(),
	}
	nodeCfg.DBPath = dbPath
	nodeCfg.ChainID = "mainnet"
	privKey, err := signlib.NewPrivKey()
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("NewPrivKey")
	}

	chain, txPool, err := mockChainAndPool()
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("mockChainAndPool")
	}

	return &monitor{
		cfg:            cfg,
		db:             db,
		nodeCfg:        nodeCfg,
		privKey:        privKey.(chainkd.XPrv),
		chain:          chain,
		txPool:         txPool,
		bestHeightSeen: uint64(0),
	}
}

func makePath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	dataPath := usr.HomeDir + "/.vapor/precog"
	if err := os.MkdirAll(dataPath, os.ModePerm); err != nil {
		return "", err
	}

	return dataPath, nil
}

func (m *monitor) Run() {
	if err := m.makeSwitch(); err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("makeSwitch")
	}

	go m.discoveryRoutine()
	go m.connectionRoutine()
}

func (m *monitor) makeSwitch() error {
	var seeds []string
	for _, node := range m.cfg.Nodes {
		seeds = append(seeds, fmt.Sprintf("%s:%d", node.IP, node.Port))
	}
	m.nodeCfg.P2P.Seeds = strings.Join(seeds, ",")

	l, listenAddr := p2p.GetListener(m.nodeCfg.P2P)
	discv, err := dht.NewDiscover(m.nodeCfg, m.privKey, l.ExternalAddress().Port, m.cfg.NetworkID)
	if err != nil {
		return err
	}

	// no need for lanDiscv, but passing &mdns.LANDiscover{} will cause NilPointer
	lanDiscv := mdns.NewLANDiscover(mdns.NewProtocol(), int(l.ExternalAddress().Port))
	m.sw, err = p2p.NewSwitch(m.nodeCfg, discv, lanDiscv, l, m.privKey, listenAddr, m.cfg.NetworkID)
	if err != nil {
		return err
	}

	m.peers = peers.NewPeerSet(m.sw)
	return m.prepareReactors()
}

func (m *monitor) prepareReactors() error {
	dispatcher := event.NewDispatcher()
	// add ConsensusReactor for consensusChannel
	_ = consensusmgr.NewManager(m.sw, m.chain, m.peers, dispatcher)
	fastSyncDB := dbm.NewDB("fastsync", m.nodeCfg.DBBackend, m.nodeCfg.DBDir())
	// add ProtocolReactor to handle msgs
	if _, err := chainmgr.NewManager(m.nodeCfg, m.sw, m.chain, m.txPool, dispatcher, m.peers, fastSyncDB); err != nil {
		return err
	}

	for label, reactor := range m.sw.GetReactors() {
		log.WithFields(log.Fields{"label": label, "reactor": reactor}).Debug("start reactor")
		if _, err := reactor.Start(); err != nil {
			return err
		}
	}

	m.sw.GetSecurity().RegisterFilter(m.sw.GetNodeInfo())
	m.sw.GetSecurity().RegisterFilter(m.sw.GetPeers())
	return m.sw.GetSecurity().Start()
}
