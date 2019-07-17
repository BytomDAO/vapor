package reward

import (
	"encoding/json"
	"fmt"

	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/toolbar/common"
)

var buildSpendReqFmt = `
	{"actions": [
		%s,
		%s
	]}`

var inputActionFmt = `
{"type": "spend_account", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","amount": %d,"account_id": "%s"}
`
var outputActionFmt = `
{"type": "control_address", "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "amount": %d, "address": "%s"}
`

type Transaction struct {
	ip string
}

func (t *Transaction) buildTx(inputAction, outputAction string) (*txbuilder.Template, error) {
	url := "/build-transaction"
	buildReqStr := fmt.Sprintf(buildSpendReqFmt, inputAction, outputAction)

	tmpl := &txbuilder.Template{}
	return tmpl, t.request(url, []byte(buildReqStr), tmpl)
}

type signTxReq struct {
	Password string             `json:"password"`
	Txs      txbuilder.Template `json:"transaction"`
}

type signTemplateResp struct {
	Tx           *txbuilder.Template `json:"transaction"`
	SignComplete bool                `json:"sign_complete"`
}

func (t *Transaction) signTx(passwd string, tmpl txbuilder.Template) (*txbuilder.Template, error) {
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

	if err := t.request(url, payload, resp); err != nil {
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

func (t *Transaction) SubmitTx(tx interface{}) (string, error) {
	url := "/submit-transaction"
	payload, err := json.Marshal(submitTxReq{Tx: tx})
	if err != nil {
		return "", errors.Wrap(err, "json marshal")
	}

	res := &submitTxResp{}
	return res.TxID, t.request(url, payload, res)
}

type response struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrDetail string          `json:"error_detail"`
}

func (t *Transaction) request(url string, payload []byte, respData interface{}) error {
	resp := &response{}
	if err := common.Post(t.ip+url, payload, resp); err != nil {
		return err
	}

	if resp.Status != "success" {
		return errors.New(resp.ErrDetail)
	}

	return json.Unmarshal(resp.Data, respData)
}

type getRawBlockReq struct {
	BlockHeight uint64 `json:"block_height"`
	BlockHash   string `json:"block_hash"`
}

type getRawBlockResp struct {
	RawBlock          *types.Block          `json:"raw_block"`
	TransactionStatus *bc.TransactionStatus `json:"transaction_status"`
}

// GetCoinbaseTx return coinbase tx
func (t *Transaction) GetCoinbaseTx(blockHeight uint64) (*types.Tx, error) {
	url := "/get-raw-block"
	req := getRawBlockReq{
		BlockHeight: blockHeight,
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &getRawBlockResp{}
	if err := t.request(url, payload, res); err != nil {
		return nil, err
	}
	if len(res.RawBlock.Transactions) > 0 {
		return res.RawBlock.Transactions[0], nil
	}

	return nil, errors.New("no coinbase")
}
