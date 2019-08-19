package monitor

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	vaporCfg "github.com/vapor/config"
	"github.com/vapor/crypto/ed25519/chainkd"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/event"
	"github.com/vapor/netsync/chainmgr"
	"github.com/vapor/netsync/consensusmgr"
	"github.com/vapor/p2p"
	// msgs "github.com/vapor/netsync/messages"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/p2p/discover/dht"
	"github.com/vapor/p2p/discover/mdns"
	"github.com/vapor/p2p/signlib"
	"github.com/vapor/test/mock"
	"github.com/vapor/toolbar/precog/config"
)

type monitor struct {
	*sync.RWMutex
	cfg       *config.Config
	db        *gorm.DB
	nodeCfg   *vaporCfg.Config
	sw        *p2p.Switch
	discvCh   chan *dht.Node
	privKey   chainkd.XPrv
	chain     *mock.Chain
	txPool    *mock.Mempool
	connected uint32
}

// TODO: set myself as SPV?
func NewMonitor(cfg *config.Config, db *gorm.DB) *monitor {
	//TODO: for test
	cfg.CheckFreqSeconds = 30

	dbPath, err := makePath()
	if err != nil {
		log.Fatal(err)
	}

	nodeCfg := &vaporCfg.Config{
		BaseConfig: vaporCfg.DefaultBaseConfig(),
		P2P:        vaporCfg.DefaultP2PConfig(),
		Federation: vaporCfg.DefaultFederationConfig(),
	}
	nodeCfg.DBPath = dbPath
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
		RWMutex: &sync.RWMutex{},
		cfg:     cfg,
		db:      db,
		nodeCfg: nodeCfg,
		discvCh: discvCh,
		privKey: privKey.(chainkd.XPrv),
		chain:   chain,
		txPool:  txPool,
	}
}

func makePath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	dataPath := usr.HomeDir + "/.precog"
	if err := os.MkdirAll(dataPath, os.ModePerm); err != nil {
		return "", err
	}

	tmpDir, err := ioutil.TempDir(dataPath, "")
	if err != nil {
		return "", err
	}

	return tmpDir, nil
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

	for label, reactor := range m.sw.GetReactors() {
		log.Debug("start reactor: (%s:%v)", label, reactor)
		if _, err := reactor.Start(); err != nil {
			return nil
		}
	}

	m.sw.GetSecurity().RegisterFilter(m.sw.GetNodeInfo())
	m.sw.GetSecurity().RegisterFilter(m.sw.GetPeers())

	return m.sw.GetSecurity().Start()
}

func (m *monitor) checkStatusRoutine() {
	peers := peers.NewPeerSet(m.sw)
	if err := m.prepareReactors(peers); err != nil {
		log.Fatal(err)
	}

	protocolReactor, ok := m.sw.GetReactors()["PROTOCOL"]
	if !ok {
		log.Fatal("protocolReactor not found")
	}

	bestHeight := uint64(0)
	ticker := time.NewTicker(time.Duration(m.cfg.CheckFreqSeconds) * time.Second)
	for range ticker.C {
		m.Lock()
		log.Info("connected peers: ", m.sw.GetPeers().List())

		for _, peer := range m.sw.GetPeers().List() {
			peer.Start()
			protocolReactor.AddPeer(peer)
		}

		for _, peer := range m.sw.GetPeers().List() {
			p := peers.GetPeer(peer.ID())
			if p == nil {
				continue
			}

			if err := p.SendStatus(m.chain.BestBlockHeader(), m.chain.LastIrreversibleHeader()); err != nil {
				peers.RemovePeer(p.ID())
			}
		}

		for _, peerInfo := range peers.GetPeerInfos() {
			if peerInfo.Height > bestHeight {
				bestHeight = peerInfo.Height
			}

			m.savePeerInfo(peerInfo)
		}
		log.Info("bestHeight: ", bestHeight)

		// TODO:
		// msg := struct{ msgs.BlockchainMessage }{&msgs.GetBlockMessage{Height: bestHeight + 1}}
		// for _, peer := range m.sw.GetPeers().List() {
		// 	peers.SendMsg(peer.ID(), msgs.BlockchainChannel, msg)
		// }

		for _, peer := range m.sw.GetPeers().List() {
			p := peers.GetPeer(peer.ID())
			if p == nil {
				continue
			}

			peers.RemovePeer(p.ID())
		}
		log.Info("Disonnect all peers.")
		m.Unlock()
	}
}
