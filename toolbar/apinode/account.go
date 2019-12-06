package apinode

import (
	"encoding/json"

	"github.com/bytom/vapor/api"
	"github.com/bytom/vapor/blockchain/pseudohsm"
	"github.com/bytom/vapor/blockchain/query"
	"github.com/bytom/vapor/blockchain/txbuilder"
	"github.com/bytom/vapor/crypto/ed25519/chainkd"
	"github.com/bytom/vapor/errors"
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
func (n *Node) CreateAccount(keyAlias, accountAlias string) (*query.AnnotatedAccount, error) {
	xPub, err := n.ListKeys()
	if err != nil {
		return nil, err
	}

	var rootXPub *chainkd.XPub
	for _, x := range *xPub {
		if x.Alias == keyAlias {
			rootXPub = &x.XPub
			break
		}
	}

	if rootXPub == nil {
		return nil, errors.New("keyAlias not found!")
	}

	return n.postCreateAccount(accountAlias, 1, []chainkd.XPub{*rootXPub})
}

//多签账户
func (n *Node) CreateMultiSignAccount(alias string, quorum int, rootXPubs []chainkd.XPub) (*query.AnnotatedAccount, error) {
	return n.postCreateAccount(alias, quorum, rootXPubs)
}

func (n *Node) postCreateAccount(alias string, quorum int, rootXPubs []chainkd.XPub) (*query.AnnotatedAccount, error) {
	url := "/create-account"
	payload, err := json.Marshal(api.CreateAccountReq{
		Alias:     alias,
		Quorum:    quorum,
		RootXPubs: rootXPubs,
	})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &query.AnnotatedAccount{}
	return res, n.request(url, payload, res)
}

func (n *Node) ListAccounts() (*[]query.AnnotatedAccount, error) {
	url := "/list-accounts"
	payload, err := json.Marshal(struct{}{})
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
