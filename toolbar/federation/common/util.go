package common

import (
	"encoding/hex"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/consensus"
	"github.com/vapor/consensus/segwit"
	"github.com/vapor/wallet"
)

func ProgToAddress(prog []byte, netParams *consensus.Params) string {
	hash, err := segwit.GetHashFromStandardProg(prog)
	if err != nil {
		log.WithFields(log.Fields{"prog": hex.EncodeToString(prog), "err": err}).Warn("fail on GetHashFromStandardProg")
		return ""
	}

	if segwit.IsP2WPKHScript(prog) {
		return wallet.BuildP2PKHAddress(hash, netParams)
	} else if segwit.IsP2WSHScript(prog) {
		return wallet.BuildP2SHAddress(hash, netParams)
	}
	return ""
}
