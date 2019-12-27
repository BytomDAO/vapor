package common

import (
	"encoding/hex"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/consensus/segwit"
	"github.com/bytom/vapor/wallet"
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
