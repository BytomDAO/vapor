package federation

import (
	log "github.com/sirupsen/logrus"

	"github.com/vapor/crypto/ed25519/chainkd"
)

var xprvStr = "d20e3d81ba2c5509619fbc276d7cd8b94f52a1dce1291ae9e6b28d4a48ee67d8ac5826ba65c9da0b035845b7cb379e816c529194c7e369492d8828dee5ede3e2"

func string2xprv(str string) (xprv chainkd.XPrv) {
	if err := xprv.UnmarshalText([]byte(str)); err != nil {
		log.Panicf("fail to convert xprv string")
	}
	return xprv
}
