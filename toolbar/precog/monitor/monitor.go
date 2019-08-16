package monitor

import (
	// "encoding/binary"
	// "encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	// "os/user"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	// dbm "github.com/vapor/database/leveldb"

	vaporCfg "github.com/vapor/config"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto/ed25519/chainkd"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/event"
	"github.com/vapor/p2p"
	"github.com/vapor/protocol/bc/types"
	// conn "github.com/vapor/p2p/connection"
	"github.com/vapor/netsync/peers"
	// "github.com/vapor/consensus"
	// "github.com/vapor/crypto/sha3pool"
	"github.com/vapor/netsync/chainmgr"
	"github.com/vapor/netsync/consensusmgr"
	"github.com/vapor/p2p/discover/dht"
	"github.com/vapor/p2p/discover/mdns"
	"github.com/vapor/p2p/signlib"
	"github.com/vapor/test/mock"
	"github.com/vapor/toolbar/precog/config"
	"github.com/vapor/toolbar/precog/database/orm"
)

var (
	nodesToDiscv = 150
	discvFreqSec = 60
)

type monitor struct {
	cfg     *config.Config
	db      *gorm.DB
	nodeCfg *vaporCfg.Config
	sw      *p2p.Switch
	discvCh chan *dht.Node
	privKey chainkd.XPrv
}

// TODO: set SF myself?
func NewMonitor(cfg *config.Config, db *gorm.DB) *monitor {
	//TODO: for test
	cfg.CheckFreqSeconds = 1

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

func (m *monitor) discoveryRoutine() {
	ticker := time.NewTicker(time.Duration(discvFreqSec) * time.Second)
	for range ticker.C {
		nodes := make([]*dht.Node, nodesToDiscv)
		n := m.sw.GetDiscv().ReadRandomNodes(nodes)
		for i := 0; i < n; i++ {
			m.discvCh <- nodes[i]
		}
	}
}

func (m *monitor) collectDiscoveredNodes() {
	// nodeMap maps a node's public key to the node itself
	nodeMap := make(map[string]*dht.Node)
	for node := range m.discvCh {
		if n, ok := nodeMap[node.ID.String()]; ok && n.String() == node.String() {
			continue
		}
		log.Info("discover new node: ", node)

		if err := m.upSertNode(&config.Node{
			PublicKey: node.ID.String(),
			Host:      node.IP.String(),
			Port:      node.TCP,
		}); err != nil {
			log.Error(err)
		}

		nodeMap[node.ID.String()] = node
	}
}

func (m *monitor) connectNodesRoutine() {
	// TODO: change name?
	ticker := time.NewTicker(time.Duration(m.cfg.CheckFreqSeconds) * time.Second)
	for ; true; <-ticker.C {
		if err := m.dialNodes(); err != nil {
			log.Error(err)
		}
	}
}

func (m *monitor) dialNodes() error {
	var nodes []*orm.Node
	if err := m.db.Model(&orm.Node{}).Find(&nodes).Error; err != nil {
		return err
	}

	addresses := make([]*p2p.NetAddress, 0)
	for i := 0; i < len(nodes); i++ {
		ips, err := net.LookupIP(nodes[i].Host)
		if err != nil {
			log.Error(err)
			continue
		}
		if len(ips) == 0 {
			log.Errorf("fail to look up ip for %s", nodes[i].Host)
			continue
		}

		address := p2p.NewNetAddressIPPort(ips[0], nodes[i].Port)
		addresses = append(addresses, address)
	}

	m.sw.DialPeers(addresses)
	return nil
}

func (m *monitor) getGenesisBlock() (*types.Block, error) {
	genesisBlock := &types.Block{}
	if err := genesisBlock.UnmarshalText([]byte("030100000000000000000000000000000000000000000000000000000000000000000082bfe3f4bf2d4052415e796436f587fac94677b20f027e910b70e2c220c411c0e87c37e0e1cc2ec9c377e5192668bc0a367e4a4764f11e7c725ecced1d7b6a492974fab1b6d5bc01000107010001012402220020f86826d640810eb08a2bfb706e0092273e05e9a7d3d71f9d53f4f6cc2e3d6c6a0001013b0039ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00011600148c9d063ff74ee6d9ffa88d83aeb038068366c4c400")); err != nil {
		return nil, err
	}

	return genesisBlock, nil
}

func (m *monitor) checkStatusRoutine() {
	txPool := &mock.Mempool{}
	mockChain := mock.NewChain(txPool)
	genesisBlock, err := m.getGenesisBlock()
	if err != nil {
		log.Fatal(err)
	}

	mockChain.SetBlockByHeight(genesisBlock.BlockHeader.Height, genesisBlock)
	mockChain.SetBestBlockHeader(&genesisBlock.BlockHeader)
	dispatcher := event.NewDispatcher()
	peers := peers.NewPeerSet(m.sw)
	// add ConsensusReactor for consensusChannel
	_ = consensusmgr.NewManager(m.sw, mockChain, peers, dispatcher)
	fastSyncDB := dbm.NewDB("fastsync", m.nodeCfg.DBBackend, m.nodeCfg.DBDir())
	// add ProtocolReactor to handle msgs
	_, err := chainmgr.NewManager(m.nodeCfg, m.sw, mockChain, txPool, dispatcher, peers, fastSyncDB)
	if err != nil {
		log.Fatal(err)
	}
	// ??
	m.sw.Start()

	// for k, v := range m.sw.GetReactors() {
	// 	log.Debug("start", k, ",", v)
	// 	v.Start()
	// }
	ticker := time.NewTicker(time.Duration(m.cfg.CheckFreqSeconds) * time.Second)
	for ; true; <-ticker.C {
		for _, v := range m.sw.GetReactors() {
			for _, peer := range m.sw.GetPeers().List() {
				log.Debug("AddPeer for", v, peer)
				// TODO: if not in sw
				v.AddPeer(peer)
			}
		}

		// TODO: SFSPV?
		log.Debug("best", peers.BestPeer(consensus.SFFullNode))
		for _, peerInfo := range peers.GetPeerInfos() {
			log.Info(peerInfo)
		}
	}
}

// TODO:
// implement logic first, and then refactor
// /home/gavin/work/go/src/github.com/vapor/
// p2p/test_util.go
// p2p/switch_test.go
// syncManager

// TODO: dial nodes
// TODO: get lantency
// TODO: get best_height
// TODO: decide check_height("best best_height" - "confirmations")
// TODO: get blockhash by check_height, get latency
// TODO: update lantency, active_time and status
