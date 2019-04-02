package dpos

import (
	"github.com/vapor/crypto/ed25519/chainkd"
	chainjson "github.com/vapor/encoding/json"
	"github.com/vapor/protocol/vm"
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

type DposMsg struct {
	Type vm.Op  `json:"type"`
	Data []byte `json:"data"`
}

// DELEGATE_IDS PUBKEY SIG(block.time)
type DelegateInfoList struct {
	Delegate DelegateInfo       `json:"delegate"`
	Xpub     chainkd.XPub       `json:"xpub"`
	SigTime  chainjson.HexBytes `json:"sig_time"`
}

func (d *DelegateInfoList) DataType() uint8 { return DelegateInfoType }

type RegisterForgerData struct {
	Name string `json:"name"`
}

func (d *RegisterForgerData) DataType() uint8 { return RegisterType }

type VoteForgerData struct {
	Forgers []string `json:"forgers"`
}

func (d *VoteForgerData) DataType() uint8 { return VoteType }

type CancelVoteForgerData struct {
	Forgers []string `json:"forgers"`
}

func (d *CancelVoteForgerData) DataType() uint8 { return CancelVoteType }
