package monitor

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/p2p/discover/dht"
	"github.com/vapor/toolbar/precog/config"
)

var (
	nodesToDiscv = 150
	discvFreqSec = 60
)

func (m *monitor) discoveryRoutine() {
	ticker := time.NewTicker(time.Duration(discvFreqSec) * time.Second)
	for range ticker.C {
		nodes := make([]*dht.Node, nodesToDiscv)
		n := m.sw.GetDiscv().ReadRandomNodes(nodes)
		m.Lock()
		for i := 0; i < n; i++ {
			m.discvCh <- nodes[i]
		}
		m.Unlock()
	}
}

func (m *monitor) collectDiscoveredNodes() {
	// nodeMap maps a node's public key to the node itself
	nodeMap := make(map[string]*dht.Node)
	for node := range m.discvCh {
		if n, ok := nodeMap[node.ID.String()]; ok && n.String() == node.String() {
			continue
		}
		log.Infof("discover new node: %v", node)

		if err := m.upSertNode(&config.Node{
			PublicKey: node.ID.String(),
			Host:      node.IP.String(),
			Port:      node.TCP,
		}); err == nil {
			nodeMap[node.ID.String()] = node
		} else {
			log.Error(err)
		}
	}
}
