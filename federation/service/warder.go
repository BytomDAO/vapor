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

// TODO:
// RequestSign() will request a remote warder to sign a tx, the remote warder
// will sign the tx, update its tx data & signs data, and response with the signs
func (w *Warder) RequestSign(destTx interface{}, ormTx *orm.CrossTransaction) (string, error) {
	return "", nil
}

// TODO:
// NotifySubmission() will notify a remote warder the submission of a destTx,
// the remote warder will check the tx signs count and set the tx submitted if
// the signs count reach quorum
func (w *Warder) NotifySubmission(ormTx *orm.CrossTransaction) error {
	return nil
}
