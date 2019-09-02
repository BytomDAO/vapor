package apinode

import (
	"encoding/json"

	"github.com/vapor/api"
	"github.com/vapor/blockchain/pseudohsm"
	"github.com/vapor/blockchain/query"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
)

func (n *Node) CreateKey(alias, password string) (*api.CreateKeyResp, error) {
	url := "/create-key"
	payload, err := json.Marshal(api.CreateKeyReq{Alias: alias, Password: password})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &api.CreateKeyResp{}
	return res, n.request(url, payload, res)
}

func (n *Node) ListKeys() (*[]pseudohsm.XPub, error) {
	url := "/list-keys"
	res := &[]pseudohsm.XPub{}
	return res, n.request(url, nil, res)
}

//默认创建单签账户
func (n *Node) CreateAccount(alias string) (*query.AnnotatedAccount, error) {
	xPub, err := n.ListKeys()
	if err != nil {
		return nil, err
	}

	rootXpub := chainkd.XPub{}
	for _, x := range *xPub {
		if x.Alias == alias {
			rootXpub = x.XPub
			break
		}
	}

	url := "/create-account"
	payload, err := json.Marshal(api.CreateAccountReq{
		Alias:     alias,
		Quorum:    1,
		RootXPubs: []chainkd.XPub{rootXpub},
	})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &query.AnnotatedAccount{}
	return res, n.request(url, payload, res)
}

func (n *Node) ListAccounts() (*[]query.AnnotatedAccount, error) {
	url := "/list-accounts"
	payload, err := json.Marshal(struct {
		ID    string `json:"id"`
		Alias string `json:"alias"`
	}{})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &[]query.AnnotatedAccount{}
	return res, n.request(url, payload, res)
}

func (n *Node) CreateAccountReceiver(alias string) (*txbuilder.Receiver, error) {
	url := "/create-account-receiver"
	payload, err := json.Marshal(api.AccountFilter{
		AccountAlias: alias,
	})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &txbuilder.Receiver{}
	return res, n.request(url, payload, res)
}
