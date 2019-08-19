package monitor

import (
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vapor/p2p"
	"github.com/vapor/toolbar/precog/database/orm"
)

func (m *monitor) connectNodesRoutine() {
	// TODO: change name?
	ticker := time.NewTicker(time.Duration(m.cfg.CheckFreqSeconds*2) * time.Second)
	for ; true; <-ticker.C {
		if err := m.dialNodes(); err != nil {
			log.Error(err)
		}
	}
}

func (m *monitor) dialNodes() error {
	// m.Lock()
	log.Info("Start to reconnect to nodes...")
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

	// connected peers will be skipped in switch.DialPeers()
	m.sw.DialPeers(addresses)
	log.Info("DialPeers done.")
	// m.Unlock()
	return nil
}
