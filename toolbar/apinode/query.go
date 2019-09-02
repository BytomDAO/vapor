package apinode

import (
	"encoding/json"

	"github.com/vapor/api"
	"github.com/vapor/blockchain/query"
	"github.com/vapor/errors"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/wallet"
)

func (n *Node) ListAddresses(accountID, accountAlias string, from, count uint) (*[]api.AddressResp, error) {
	url := "/list-addresses"
	payload, err := json.Marshal(api.AddressReq{
		AccountID:    accountID,
		AccountAlias: accountAlias,
		From:         from,
		Count:        count,
	})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &[]api.AddressResp{}
	return res, n.request(url, payload, res)
}

func (n *Node) ListBalances(accountID, accountAlias string) (*[]wallet.AccountBalance, error) {
	url := "/list-balances"
	payload, err := json.Marshal(struct {
		AccountID    string `json:"account_id"`
		AccountAlias string `json:"account_alias"`
	}{
		AccountID:    accountID,
		AccountAlias: accountAlias,
	})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &[]wallet.AccountBalance{}
	return res, n.request(url, payload, res)
}

func (n *Node) ListUtxos(accountID, accountAlias, id string, unconfirmed, smartContract bool, from, count uint) (*[]query.AnnotatedUTXO, error) {
	url := "/list-unspent-outputs"
	payload, err := json.Marshal(api.ListUtxosReq{
		AccountID:     accountID,
		AccountAlias:  accountAlias,
		ID:            id,
		Unconfirmed:   unconfirmed,
		SmartContract: smartContract,
		From:          from,
		Count:         count,
	})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &[]query.AnnotatedUTXO{}
	return res, n.request(url, payload, res)
}

func (n *Node) WalletInfo() (*api.WalletInfo, error) {
	url := "/wallet-info"
	res := &api.WalletInfo{}
	return res, n.request(url, nil, res)
}

func (n *Node) NetInfo() (*api.NetInfo, error) {
	url := "/net-info"
	res := &api.NetInfo{}
	return res, n.request(url, nil, res)
}

func (n *Node) ListPeers() (*[]*peers.PeerInfo, error) {
	url := "/list-peers"
	res := &[]*peers.PeerInfo{}
	return res, n.request(url, nil, res)
}
