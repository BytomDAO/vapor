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
		m.Lock()

		nodes := make([]*dht.Node, nodesToDiscv)
		num := m.sw.GetDiscv().ReadRandomNodes(nodes)
		for _, node := range nodes[:num] {
			if n, ok := m.discvMap[node.ID.String()]; ok && n.String() == node.String() {
				continue
			}

			log.Infof("discover new node: %v", node)
			m.saveDiscoveredNode(node)
		}

		m.Unlock()
	}
}

func (m *monitor) saveDiscoveredNode(node *dht.Node) {
	if err := m.upSertNode(&config.Node{
		PublicKey: node.ID.String(),
		IP:        node.IP.String(),
		Port:      node.TCP,
	}); err == nil {
		m.discvMap[node.ID.String()] = node
	} else {
		log.Error(err)
	}
}
