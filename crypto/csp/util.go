package csp

import (
	"io"

	"github.com/vapor/crypto/ed25519"
	"github.com/vapor/crypto/ed25519/chainkd"
)

// Utility functions

func NewXKeys(r io.Reader) (xprv chainkd.XPrv, xpub chainkd.XPub, err error) {
	xprv, err = chainkd.NewXPrv(r)
	if err != nil {
		return
	}
	return xprv, xprv.XPub(), nil
}

func XPubKeys(xpubs []chainkd.XPub) []ed25519.PublicKey {
	res := make([]ed25519.PublicKey, 0, len(xpubs))
	for _, xpub := range xpubs {
		res = append(res, xpub.PublicKey())
	}
	return res
}

func DeriveXPubs(xpubs []chainkd.XPub, path [][]byte) []chainkd.XPub {
	res := make([]chainkd.XPub, 0, len(xpubs))
	for _, xpub := range xpubs {
		d := xpub.Derive(path)
		res = append(res, d)
	}
	return res
}
