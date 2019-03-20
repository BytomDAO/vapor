package consensus

import (
	"github.com/vapor/crypto/ed25519/chainkd"
	chainjson "github.com/vapor/encoding/json"
)

// serflag variables for input types.
const (
	DelegateInfoType uint8 = iota
	RegisterType
	VoteType
	CancelVoteType
)

type TypedData interface {
	DataType() uint8
}

// DELEGATE_IDS PUBKEY SIG(block.time)
type DelegateInfoList struct {
	Delegate DelegateInfo
	Xpub     chainkd.XPub
	SigTime  []chainjson.HexBytes `json:"sig_time"`
}

func (d *DelegateInfoList) DataType() uint8 { return DelegateInfoType }

type RegisterForgerData struct {
	Name string `json:"name"`
}

func (d *RegisterForgerData) DataType() uint8 { return RegisterType }

type VoteForgerData struct {
	Forgers []string `json:"Forgers"`
}

func (d *VoteForgerData) DataType() uint8 { return VoteType }

type CancelVoteForgerData struct {
	Forgers []string `json:"Forgers"`
}

func (d *CancelVoteForgerData) DataType() uint8 { return CancelVoteType }
