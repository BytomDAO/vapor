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

func (m *monitor) discoveryRoutine( /*discvWg *sync.WaitGroup*/ ) {
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
	for node := range m.discvCh {
		if n, ok := m.nodeMap[node.ID.String()]; ok && n.String() == node.String() {
			continue
		}
		log.Infof("discover new node: %v", node)

		if err := m.upSertNode(&config.Node{
			PublicKey: node.ID.String(),
			Host:      node.IP.String(),
			Port:      node.TCP,
		}); err == nil {
			m.nodeMap[node.ID.String()] = node
		} else {
			log.Error(err)
		}
	}
}
