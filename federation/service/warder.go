package service

import (
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/federation/config"
)

type Warder struct {
	hostPort string
	position uint8
	xpub     chainkd.XPub
}

func NewWarder(cfg *config.Warder) *Warder {
	return &Warder{
		hostPort: cfg.HostPort,
		position: cfg.Position,
		xpub:     cfg.XPub,
	}
}
