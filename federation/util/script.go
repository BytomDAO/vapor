package util

import (
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/crypto"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/federation/config"
	"github.com/vapor/protocol/vm/vmutil"
)

func SegWitWrap(script []byte) []byte {
	scriptHash := crypto.Sha256(script)
	wscript, err := vmutil.P2WSHProgram(scriptHash)
	if err != nil {
		log.Panicf("Fail converts scriptHash to witness: %v", err)
	}

	return wscript
}

func ParseFedProg(warders []config.Warder, quorum int) []byte {
	SortWarders(warders)

	xpubs := []chainkd.XPub{}
	for _, w := range warders {
		xpubs = append(xpubs, w.XPub)
	}

	fedScript, err := vmutil.P2SPMultiSigProgram(chainkd.XPubKeys(xpubs), quorum)
	if err != nil {
		log.Panicf("fail to generate federation scirpt for federation: %v", err)
	}

	return fedScript
}

type byPosition []config.Warder

func (w byPosition) Len() int           { return len(w) }
func (w byPosition) Swap(i, j int)      { w[i], w[j] = w[j], w[i] }
func (w byPosition) Less(i, j int) bool { return w[i].Position < w[j].Position }

func SortWarders(warders []config.Warder) []config.Warder {
	sort.Sort(byPosition(warders))
	return warders
}
