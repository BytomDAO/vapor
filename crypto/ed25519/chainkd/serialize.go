package chainkd

import (
	"encoding/hex"
	"errors"
	"fmt"
)

const (
	extendedPublicKeySize  = 64
	extendedPrivateKeySize = 64
)

var (
	ErrBadKeyLen = errors.New("bad key length")
	ErrBadKeyStr = errors.New("bad key string")
)

func (xpub XPub) MarshalText() ([]byte, error) {
	hexBytes := make([]byte, hex.EncodedLen(len(xpub.Bytes())))
	hex.Encode(hexBytes, xpub.Bytes())
	return hexBytes, nil
}

func (xpub XPub) Bytes() []byte {
	return xpub[:]
}

func (xprv XPrv) MarshalText() ([]byte, error) {
	hexBytes := make([]byte, hex.EncodedLen(len(xprv.Bytes())))
	hex.Encode(hexBytes, xprv.Bytes())
	return hexBytes, nil
}

func (xprv XPrv) Bytes() []byte {
	return xprv[:]
}

func (xpub *XPub) UnmarshalText(inp []byte) error {
	if len(inp) != 2*extendedPublicKeySize {
		return ErrBadKeyStr
	}
	_, err := hex.Decode(xpub[:], inp)
	return err
}

func (xpub XPub) String() string {
	return hex.EncodeToString(xpub.Bytes())
}

func (xprv *XPrv) UnmarshalText(inp []byte) error {
	if len(inp) != 2*extendedPrivateKeySize {
		return ErrBadKeyStr
	}
	_, err := hex.Decode(xprv[:], inp)
	return err
}

func (xprv XPrv) String() string {
	return hex.EncodeToString(xprv.Bytes())
}

func NewXPub(str string) (xpub *XPub, err error) {
	if len(str) != 2*extendedPublicKeySize {
		fmt.Println("str length is:", len(str))
		fmt.Println("str is:", str)
		return nil, errors.New("string length is invalid.")
	}
	if xpubBytes, err := hex.DecodeString(str); err != nil {
		return nil, err
	} else {
		copy(xpub[:], xpubBytes[:])
	}

	return xpub, nil
}
