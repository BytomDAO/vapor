package service

import (
	"encoding/json"
	"fmt"

	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc/types"
)

var buildSpendReqFmt = `
	{"actions": [
		%s,
		%s
	]}`

var InputActionFmt = `
{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount": %d,"account_id": "%s"}
`
var OutputActionFmt = `
{"type": "control_address", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount": %d, "address": "%s"}
`

func (n *Node) SendTransaction(inputAction, outputAction, passwd string) (string, error) {
	tmpl, err := n.buildTx(inputAction, outputAction)
	if err != nil {
		return "", err
	}

	tmpl, err = n.signTx(passwd, *tmpl)
	if err != nil {
		return "", err
	}

	return n.SubmitTx(tmpl.Transaction)
}

func (n *Node) buildTx(inputAction, outputAction string) (*txbuilder.Template, error) {
	url := "/build-transaction"
	buildReqStr := fmt.Sprintf(buildSpendReqFmt, inputAction, outputAction)

	tmpl := &txbuilder.Template{}
	return tmpl, n.request(url, []byte(buildReqStr), tmpl)
}

type signTxReq struct {
	Password string             `json:"password"`
	Txs      txbuilder.Template `json:"transaction"`
}

type signTemplateResp struct {
	Tx           *txbuilder.Template `json:"transaction"`
	SignComplete bool                `json:"sign_complete"`
}

func (n *Node) signTx(passwd string, tmpl txbuilder.Template) (*txbuilder.Template, error) {
	url := "/sign-transaction"
	req := &signTxReq{
		Password: passwd,
		Txs:      tmpl,
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	resp := &signTemplateResp{}

	if err := n.request(url, payload, resp); err != nil {
		return nil, err
	}

	if !resp.SignComplete {
		return nil, errors.Wrap(err, "sign fail")
	}

	return resp.Tx, nil
}

type submitTxReq struct {
	Tx interface{} `json:"raw_transaction"`
}

type submitTxResp struct {
	TxID string `json:"tx_id"`
}

func (n *Node) SubmitTx(tx interface{}) (string, error) {
	url := "/submit-transaction"
	payload, err := json.Marshal(submitTxReq{Tx: tx})
	if err != nil {
		return "", errors.Wrap(err, "json marshal")
	}

	res := &submitTxResp{}
	return res.TxID, n.request(url, payload, res)
}

// GetCoinbaseTx return coinbase tx
func (n *Node) GetCoinbaseTx(blockHeight uint64) (*types.Tx, error) {
	req := &getRawBlockReq{
		BlockHeight: blockHeight,
	}

	blockStr, _, err := n.getRawBlock(req)
	if err != nil {
		return nil, errors.Wrap(err, "get RawBlock")
	}

	block := &types.Block{}

	if err := block.UnmarshalText([]byte(blockStr)); err != nil {
		return nil, errors.Wrap(err, "json unmarshal block")
	}

	if len(block.Transactions) > 0 {
		return block.Transactions[0], nil
	}

	return nil, errors.New("no coinbase")
}
