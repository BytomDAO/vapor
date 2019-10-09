package mdns

import (
	"context"
	"fmt"

	"github.com/grandcat/zeroconf"
)

const (
	instanceName = "vapord"
	serviceName  = "vapor%sLanDiscover"
	domainName   = "local"
)

// Protocol decoration ZeroConf,which is a pure Golang library
// that employs Multicast DNS-SD.
type Protocol struct {
	entries     chan *zeroconf.ServiceEntry
	server      *zeroconf.Server
	serviceName string
	quite       chan struct{}
}

// NewProtocol create a specific Protocol.
func NewProtocol(chainID string) *Protocol {
	return &Protocol{
		entries:     make(chan *zeroconf.ServiceEntry),
		serviceName: fmt.Sprintf(serviceName, chainID),
		quite:       make(chan struct{}),
	}
}

func (p *Protocol) getLanPeerLoop(event chan LANPeerEvent) {
	for {
		select {
		case entry := <-p.entries:
			event <- LANPeerEvent{IP: entry.AddrIPv4, Port: entry.Port}
		case <-p.quite:
			return
		}
	}
}

func (p *Protocol) registerService(port int) error {
	var err error
	p.server, err = zeroconf.Register(instanceName, p.serviceName, domainName, port, nil, nil)
	return err
}

func (m *Protocol) registerResolver(event chan LANPeerEvent) error {
	go m.getLanPeerLoop(event)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return err
	}

	return resolver.Browse(context.Background(), m.serviceName, domainName, m.entries)
}

func (p *Protocol) stopResolver() {
	close(p.quite)
}

func (p *Protocol) stopService() {
	if p.server != nil {
		p.server.Shutdown()
	}
}
