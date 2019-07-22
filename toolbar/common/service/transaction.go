package service

import (
	"encoding/json"

	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
)

type buildSpendReq struct {
	Actions []interface{} `json:"actions"`
}

type Action struct {
	InputAction
	OutputActions []OutputAction
}

type InputAction struct {
	Type      string `json:"type"`
	AccountID string `json:"account_id"`
	bc.AssetAmount
}

type OutputAction struct {
	Type    string `json:"type"`
	Address string `json:"address"`
	bc.AssetAmount
}

func (n *Node) SendTransaction(inputAction InputAction, outputActions []OutputAction, passwd string) (string, error) {
	tmpl, err := n.buildTx(inputAction, outputActions)
	if err != nil {
		return "", err
	}

	tmpl, err = n.signTx(passwd, *tmpl)
	if err != nil {
		return "", err
	}

	return n.SubmitTx(tmpl.Transaction)
}

func (n *Node) buildRequest(inputAction InputAction, outputActions []OutputAction, req *buildSpendReq) error {
	if len(outputActions) == 0 {
		return errors.New("output is empty")
	}
	req.Actions = append(req.Actions, &inputAction)

	for _, outputAction := range outputActions {
		req.Actions = append(req.Actions, &outputAction)
	}

	return nil
}

func (n *Node) buildTx(inputAction InputAction, outputActions []OutputAction) (*txbuilder.Template, error) {
	url := "/build-transaction"

	req := &buildSpendReq{}
	err := n.buildRequest(inputAction, outputActions, req)
	if err != nil {
		return nil, errors.Wrap(err, "build spend request")
	}

	buildReq, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "Marshal spend request")
	}

	tmpl := &txbuilder.Template{}
	return tmpl, n.request(url, buildReq, tmpl)
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
