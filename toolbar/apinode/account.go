package apinode

import (
	"encoding/json"

	"github.com/vapor/api"
	"github.com/vapor/blockchain/query"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
)

func (n *Node) CreateKey(alias, password, mnemonic, language string) (*api.CreateKeyResp, error) {
	url := "/create-key"
	payload, err := json.Marshal(api.CreateKeyReq{Alias: alias, Password: password, Mnemonic: mnemonic, Language: language})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &api.CreateKeyResp{}
	return res, n.request(url, payload, res)
}

func (n *Node) CreateAccount(rootXPubs []chainkd.XPub, quorum int, alias string) (*query.AnnotatedAccount, error) {
	url := "/create-account"
	payload, err := json.Marshal(api.CreateAccountReq{RootXPubs: rootXPubs, Quorum: quorum, Alias: alias})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &query.AnnotatedAccount{}
	return res, n.request(url, payload, res)
}

func (n *Node) ListAccounts(id, alias string) (*[]query.AnnotatedAccount, error) {
	url := "/list-accounts"
	payload, err := json.Marshal(struct {
		ID    string `json:"id"`
		Alias string `json:"alias"`
	}{
		ID:    id,
		Alias: alias,
	})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &[]query.AnnotatedAccount{}
	return res, n.request(url, payload, res)
}

func (n *Node) CreateAccountReceiver(id, alias string) (*txbuilder.Receiver, error) {
	url := "/create-account-receiver"
	payload, err := json.Marshal(struct {
		AccountID    string `json:"account_id"`
		AccountAlias string `json:"account_alias"`
	}{
		AccountID:    id,
		AccountAlias: alias,
	})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &txbuilder.Receiver{}
	return res, n.request(url, payload, res)
}
