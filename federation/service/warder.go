package service

import (
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/federation/config"
)

type Warder struct {
	HostPort string
	Position uint8
	XPub     chainkd.XPub
}

func NewWarder(cfg *config.Warder) *Warder {
	return &Warder{
		HostPort: cfg.HostPort,
		Position: cfg.Position,
		XPub:     cfg.XPub,
	}
}
