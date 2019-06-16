package service

import (
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database/orm"
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

func (w *Warder) RequestSign(ormTx *orm.CrossTransaction) (string, error) {
	return "", nil
}
