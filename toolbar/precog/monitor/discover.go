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
	// discvMap maps a node's public key to the node itself
	discvMap := make(map[string]*dht.Node)
	ticker := time.NewTicker(time.Duration(discvFreqSec) * time.Second)
	for range ticker.C {
		nodes := make([]*dht.Node, nodesToDiscv)
		num := m.sw.GetDiscv().ReadRandomNodes(nodes)
		for _, node := range nodes[:num] {
			if n, ok := discvMap[node.ID.String()]; ok && n.String() == node.String() {
				continue
			}

			log.WithFields(log.Fields{"new node": node}).Info("discover")

			if err := m.upsertNode(&config.Node{
				PublicKey: node.ID.String(),
				IP:        node.IP.String(),
				Port:      node.TCP,
			}); err != nil {
				log.WithFields(log.Fields{"node": node, "err": err}).Error("upsertNode")
			} else {
				discvMap[node.ID.String()] = node
			}
		}
	}
}
