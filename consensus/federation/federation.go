package federation

import (
	"crypto"
	"encoding/json"
	"errors"
	"reflect"

	log "github.com/sirupsen/logrus"

	vcrypto "github.com/vapor/crypto"
	"github.com/vapor/crypto/csp"
	edchainkd "github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/protocol/vm/vmutil"
)

const fedCfgJson = `
{
    "xpubs" : [
    	"7f23aae65ee4307c38d342699e328f21834488e18191ebd66823d220b5a58303496c9d09731784372bade78d5e9a4a6249b2cfe2e3a85464e5a4017aa5611e47",
    	"585e20143db413e45fbc82f03cb61f177e9916ef1df0012daa8cbf6dbb1025ce8f98e51ae319327b63505b64fdbbf6d36ef916d79e6dd67d51b0bfe76fe544c5",
    	"b58170b51ca61604028ba1cb412377dfc2bc6567c0afc84c83aae1c0c297d0227ccf568561df70851f4144bbf069b525129f2434133c145e35949375b22a6c9d",
    	"983705ae71949c1a5d0fcf953658dd9ecc549f02c63e197b4d087ae31148097ece816bbc60d9012c316139fc550fa0f4b00beb0887f6b152f7a69bc8f392b9fa",
    	"d72fb92fa13bf3e0deb39de3a47c8d6eef5584719f7877c82a4c009f78fddf924d9706d48f15b2c782ec80b6bdd621a1f7ba2a0044b0e6f92245de9436885cb9",
    	"6798460919e8dc7095ee8b9f9d65033ef3da8c2334813149da5a1e52e9c6da07ba7d0e7379baaa0c8bdcb21890a54e6b7290bee077c645ee4b74b0c1ae9da59a"
    ],
    "quorum" : 4
}
`

type federation struct {
	XPubs          []vcrypto.XPubKeyer `json:"xpubs"`
	Quorum         int                 `json:"quorum"`
	ControlProgram []byte
}

func parseFedConfig() *federation {
	fed := &federation{}
	if err := json.Unmarshal([]byte(fedCfgJson), fed); err != nil {
		log.Fatalln("invalid federation config json")
	}

	for i, xpub := range fed.XPubs {
		if reflect.TypeOf(xpub).String() == "string" {
			if xpb, err := edchainkd.NewXPub(reflect.ValueOf(xpub).String()); err != nil {
				panic(err)
			} else {
				fed.XPubs[i] = *xpb
			}
		}
	}

	return fed
}

// CheckFedConfig checks the high-level rule for federation config, whereas
// signers.Create checks the low-level rule
func CheckFedConfig() error {
	fed := parseFedConfig()
	if len(fed.XPubs) <= 1 {
		return errors.New("federation should have more than 1 member")
	}
	if fed.Quorum < 1 {
		return errors.New("federation quorum should be >= 1")
	}

	return nil
}

func GetFederation() *federation {
	fed := parseFedConfig()
	PublicKeys := csp.XPubKeys(fed.XPubs)
	if controlProgram, err := fed.buildPegInControlProgram(PublicKeys); err == nil {
		fed.ControlProgram = controlProgram
	} else {
		log.Fatalf("fail to build peg-in script: %v", err)
	}

	return fed
}

func (f *federation) buildPegInControlProgram(pubkeys []crypto.PublicKey) (program []byte, err error) {
	controlProg, err := vmutil.P2SPMultiSigProgram(pubkeys, f.Quorum)
	if err != nil {
		return nil, err
	}

	builder := vmutil.NewBuilder()
	builder.AddRawBytes(controlProg)
	return builder.Build()
}
