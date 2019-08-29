package monitor

import (
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vapor/p2p"
	"github.com/vapor/toolbar/precog/database/orm"
)

func (m *monitor) connectNodesRoutine() {
	// TODO: fix
	// ticker := time.NewTicker(time.Duration(m.cfg.CheckFreqMinutes) * time.Minute)
	ticker := time.NewTicker(time.Duration(m.cfg.CheckFreqMinutes) * time.Second)
	for ; true; <-ticker.C {
		<-m.dialCh
		m.Lock()

		if err := m.dialNodes(); err != nil {
			log.WithFields(log.Fields{"err": err}).Error("dialNodes")
		}
	}
}

func (m *monitor) dialNodes() error {
	log.Info("Start to reconnect to nodes...")
	var nodes []*orm.Node
	if err := m.db.Model(&orm.Node{}).Find(&nodes).Error; err != nil {
		return err
	}

	addresses := make([]*p2p.NetAddress, 0)
	for i := 0; i < len(nodes); i++ {
		address := p2p.NewNetAddressIPPort(net.ParseIP(nodes[i].IP), nodes[i].Port)
		addresses = append(addresses, address)
	}

	// connected peers will be skipped in switch.DialPeers()
	m.sw.DialPeers(addresses)
	log.Info("DialPeers done.")
	m.processDialResults()
	m.checkStatusCh <- struct{}{}
	return nil
}
