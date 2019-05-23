package chainkd

import (
	"fmt"
	"io"

	"github.com/vapor/crypto/ed25519"
)

// Utility functions

func NewXKeys(r io.Reader) (xprv XPrv, xpub XPub, err error) {
	xprv, err = NewXPrv(r)
	if err != nil {
		return
	}
	if xpubkey, ok := xprv.XPub().(XPub); ok {
		return xprv, xpubkey, nil
	} else {
		fmt.Println("create xpubkey failed.")
	}
	return xprv, xpub, nil
}

func XPubKeys(xpubs []XPub) []ed25519.PublicKey {
	res := make([]ed25519.PublicKey, 0, len(xpubs))
	for _, xpub := range xpubs {
		res = append(res, xpub.PublicKey())
	}
	return res
}

func DeriveXPubs(xpubs []XPub, path [][]byte) []XPub {
	res := make([]XPub, 0, len(xpubs))
	for _, xpub := range xpubs {
		d := xpub.Derive(path)
		if xpk, ok := d.(XPub); ok {
			res = append(res, xpk)
		}
	}
	return res
}
