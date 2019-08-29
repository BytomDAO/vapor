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

			log.WithFields(log.Fields{"new node": node}).Info("discover")

			if err := m.upSertNode(&config.Node{
				PublicKey: node.ID.String(),
				IP:        node.IP.String(),
				Port:      node.TCP,
			}); err != nil {
				log.WithFields(log.Fields{"node": node, "err": err}).Error("upSertNode")
			} else {
				m.discvMap[node.ID.String()] = node
			}
		}

		m.Unlock()
	}
}
