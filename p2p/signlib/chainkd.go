package signlib

import (
	"errors"

	"github.com/bytom/vapor/crypto/ed25519/chainkd"
)

const (
	PubkeySize     = 64
	AuthSigMsgSize = 132
)

var (
	ErrPubkeyLength = errors.New("pubkey length error")
)

type PubKey interface {
	String() string
	Bytes() []byte
	Verify(msg []byte, sig []byte) bool
	MarshalText() ([]byte, error)
}

type PrivKey interface {
	Bytes() []byte
	Sign(msg []byte) []byte
	XPub() chainkd.XPub
}

func NewPrivKey() (PrivKey, error) {
	return chainkd.NewXPrv(nil)
}

func NewPubKey(pubkey []byte) (PubKey, error) {
	if len(pubkey) != PubkeySize {
		return nil, ErrPubkeyLength
	}

	var pubKey chainkd.XPub
	copy(pubKey[:], pubkey[:])
	return pubKey, nil
}
