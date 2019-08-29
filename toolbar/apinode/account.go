package apinode

import (
	"encoding/json"

	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
)

type createKeyReq struct {
	Alias    string `json:"alias"`
	Password string `json:"password"`
	Mnemonic string `json:"mnemonic"`
	Language string `json:"language"`
}

type createKeyResp struct {
	Alias    string       `json:"alias"`
	XPub     chainkd.XPub `json:"xpub"`
	File     string       `json:"file"`
	Mnemonic string       `json:"mnemonic,omitempty"`
}

func (n *Node) CreateKey(alias, password, mnemonic, language string) (*createKeyResp, error) {
	url := "/create-key"
	payload, err := json.Marshal(createKeyReq{Alias: alias, Password: password, Mnemonic: mnemonic, Language: language})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &createKeyResp{}
	return res, n.request(url, payload, res)
}

type createAccountReq struct {
	RootXPubs []chainkd.XPub `json:"root_xpubs"`
	Quorum    int            `json:"quorum"`
	Alias     string         `json:"alias"`
}

type createAccountResp struct {
	ID         string         `json:"id"`
	Alias      string         `json:"alias,omitempty"`
	XPubs      []chainkd.XPub `json:"xpubs"`
	Quorum     int            `json:"quorum"`
	KeyIndex   uint64         `json:"key_index"`
	DeriveRule uint8          `json:"derive_rule"`
}

func (n *Node) CreateAccount(rootXPubs []chainkd.XPub, quorum int, alias string) (*createAccountResp, error) {
	url := "/create-account"
	payload, err := json.Marshal(createAccountReq{RootXPubs: rootXPubs, Quorum: quorum, Alias: alias})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &createAccountResp{}
	return res, n.request(url, payload, res)
}
