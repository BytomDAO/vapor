package testutil

import (
	"github.com/vapor/crypto"
	"github.com/vapor/crypto/csp"
	"github.com/vapor/crypto/ed25519"
)

var (
	TestXPub   crypto.XPubKeyer
	TestXPrv   crypto.XPrvKeyer
	TestEdPub  ed25519.PublicKey
	TestEdPubs []ed25519.PublicKey
)

type zeroReader struct{}

func (z zeroReader) Read(buf []byte) (int, error) {
	for i := range buf {
		buf[i] = 0
	}
	return len(buf), nil
}

func init() {
	var err error
	_, TestXPub, err := csp.NewXKeys(zeroReader{})
	if err != nil {
		panic(err)
	}
	TestPub := TestXPub.PublicKey()
	switch tpk := TestPub.(type) {
	case ed25519.PublicKey:
		TestEdPubs = []ed25519.PublicKey{tpk}
	}
}
