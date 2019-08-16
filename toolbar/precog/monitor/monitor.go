package monitor

import (
	// "encoding/binary"
	// "encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	// "os/user"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	// dbm "github.com/vapor/database/leveldb"

	vaporCfg "github.com/vapor/config"
	"github.com/vapor/crypto/ed25519/chainkd"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/event"
	"github.com/vapor/p2p"
	// conn "github.com/vapor/p2p/connection"
	"github.com/vapor/netsync/chainmgr"
	"github.com/vapor/netsync/consensusmgr"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/p2p/discover/dht"
	"github.com/vapor/p2p/discover/mdns"
	"github.com/vapor/p2p/signlib"
	"github.com/vapor/test/mock"
	"github.com/vapor/toolbar/precog/config"
)

type monitor struct {
	cfg     *config.Config
	db      *gorm.DB
	nodeCfg *vaporCfg.Config
	sw      *p2p.Switch
	discvCh chan *dht.Node
	privKey chainkd.XPrv
	chain   *mock.Chain
	txPool  *mock.Mempool
}

// TODO: set myself as SPV?
func NewMonitor(cfg *config.Config, db *gorm.DB) *monitor {
	//TODO: for test
	cfg.CheckFreqSeconds = 1

	// TODO: fix dir
	tmpDir, err := ioutil.TempDir(".", "vpPrecog")
	if err != nil {
		log.Fatalf("failed to create temporary data folder: %v", err)
	}

	nodeCfg := &vaporCfg.Config{
		BaseConfig: vaporCfg.DefaultBaseConfig(),
		P2P:        vaporCfg.DefaultP2PConfig(),
		Federation: vaporCfg.DefaultFederationConfig(),
	}
	nodeCfg.DBPath = tmpDir
	nodeCfg.ChainID = "mainnet"
	discvCh := make(chan *dht.Node)
	privKey, err := signlib.NewPrivKey()
	if err != nil {
		log.Fatal(err)
	}

	chain, txPool, err := mockChainAndPool()
	if err != nil {
		log.Fatal(err)
	}

	return &monitor{
		cfg:     cfg,
		db:      db,
		nodeCfg: nodeCfg,
		discvCh: discvCh,
		privKey: privKey.(chainkd.XPrv),
		chain:   chain,
		txPool:  txPool,
	}
}

func (m *monitor) Run() {
	defer os.RemoveAll(m.nodeCfg.DBPath)

	var seeds []string
	for _, node := range m.cfg.Nodes {
		seeds = append(seeds, fmt.Sprintf("%s:%d", node.Host, node.Port))
		if err := m.upSertNode(&node); err != nil {
			log.Error(err)
		}
	}
	m.nodeCfg.P2P.Seeds = strings.Join(seeds, ",")
	if err := m.makeSwitch(); err != nil {
		log.Fatal(err)
	}

	go m.discoveryRoutine()
	go m.collectDiscoveredNodes()
	go m.connectNodesRoutine()
	go m.checkStatusRoutine()
}

func (m *monitor) makeSwitch() error {
	l, listenAddr := p2p.GetListener(m.nodeCfg.P2P)
	discv, err := dht.NewDiscover(m.nodeCfg, m.privKey, l.ExternalAddress().Port, m.cfg.NetworkID)
	if err != nil {
		return err
	}

	// no need for lanDiscv, but passing &mdns.LANDiscover{} will cause NilPointer
	lanDiscv := mdns.NewLANDiscover(mdns.NewProtocol(), int(l.ExternalAddress().Port))
	sw, err := p2p.NewSwitch(m.nodeCfg, discv, lanDiscv, l, m.privKey, listenAddr, m.cfg.NetworkID)
	if err != nil {
		return err
	}

	m.sw = sw
	return nil
}

func (m *monitor) prepareReactors(peers *peers.PeerSet) error {
	dispatcher := event.NewDispatcher()
	// add ConsensusReactor for consensusChannel
	_ = consensusmgr.NewManager(m.sw, m.chain, peers, dispatcher)
	fastSyncDB := dbm.NewDB("fastsync", m.nodeCfg.DBBackend, m.nodeCfg.DBDir())
	// add ProtocolReactor to handle msgs
	_, err := chainmgr.NewManager(m.nodeCfg, m.sw, m.chain, m.txPool, dispatcher, peers, fastSyncDB)
	if err != nil {
		return err
	}

	// TODO: clean up?? only start reactors??
	m.sw.Start()

	// for label, reactor := range m.sw.GetReactors() {
	// 	log.Debug("start reactor: (%s:%v)", label, reactor)
	// 	if _, err := reactor.Start(); err != nil {
	// 		return
	// 	}
	// }

	// m.sw.GetSecurity().RegisterFilter(m.sw.GetNodeInfo())
	// m.sw.GetSecurity().RegisterFilter(m.sw.GetPeers())
	// if err := m.sw.GetSecurity().Start(); err != nil {
	// 	return
	// }

	return nil
}

func (m *monitor) checkStatusRoutine() {
	peers := peers.NewPeerSet(m.sw)
	if err := m.prepareReactors(peers); err != nil {
		log.Fatal(err)
	}

	ticker := time.NewTicker(time.Duration(m.cfg.CheckFreqSeconds) * time.Second)
	for ; true; <-ticker.C {
		for _, reactor := range m.sw.GetReactors() {
			for _, peer := range m.sw.GetPeers().List() {
				log.Debug("AddPeer %v for reactor %v", peer, reactor)
				// TODO: if not in sw
				reactor.AddPeer(peer)
			}
		}

		for _, peerInfo := range peers.GetPeerInfos() {
			log.Info(peerInfo)
		}
	}
}
