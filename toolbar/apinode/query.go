package apinode

import (
	"encoding/json"

	"github.com/bytom/vapor/api"
	"github.com/bytom/vapor/blockchain/query"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/netsync/peers"
	"github.com/bytom/vapor/wallet"
)

func (n *Node) ListAddresses(accountAlias string, from, count uint) (*[]api.AddressResp, error) {
	url := "/list-addresses"
	payload, err := json.Marshal(api.AddressReq{
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

func (n *Node) ListBalances(accountAlias string) (*[]wallet.AccountBalance, error) {
	url := "/list-balances"
	payload, err := json.Marshal(api.AccountFilter{
		AccountAlias: accountAlias,
	})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &[]wallet.AccountBalance{}
	return res, n.request(url, payload, res)
}

func (n *Node) ListUtxos(accountAlias string,from, count uint) (*[]query.AnnotatedUTXO, error) {
	url := "/list-unspent-outputs"
	payload, err := json.Marshal(api.ListUtxosReq{
		AccountAlias:  accountAlias,
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
