package monitor

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/p2p/discover/dht"
	"github.com/vapor/toolbar/precog/config"
)

var (
	nodesToDiscv = 150
	discvFreqSec = 60
)

func (m *monitor) discoveryRoutine(discvWg *sync.WaitGroup) {
	ticker := time.NewTicker(time.Duration(discvFreqSec) * time.Second)
	for range ticker.C {
		m.Lock()
		nodes := make([]*dht.Node, nodesToDiscv)
		num := m.sw.GetDiscv().ReadRandomNodes(nodes)
		for i := 0; i < num; i++ {
			if n, ok := m.discvMap[nodes[i].ID.String()]; ok && n.String() == nodes[i].String() {
				continue
			}

			log.Infof("discover new node: %v", nodes[i])
			discvWg.Add(1)
			m.discvCh <- nodes[i]
		}
		discvWg.Wait()
		m.Unlock()
	}
}

func (m *monitor) collectDiscoveredNodes(discvWg *sync.WaitGroup) {
	for node := range m.discvCh {
		if err := m.upSertNode(&config.Node{
			PublicKey: node.ID.String(),
			Host:      node.IP.String(),
			Port:      node.TCP,
		}); err == nil {
			m.discvMap[node.ID.String()] = node
		} else {
			log.Error(err)
		}
		discvWg.Done()
	}
}
