package consensus

import (
	// "encoding/binary"
	// "strings"

	// "github.com/vapor/protocol/bc"
	"github.com/vapor/crypto/ed25519/chainkd"
)

type Federation struct {
	XPubs  []chainkd.XPub
	Quorum int
}

const (
	FedXPubs = ""
	Quorum   = 1
)
