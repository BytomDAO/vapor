package apinode

import (
	"encoding/json"

	"github.com/bytom/vapor/blockchain/txbuilder"
	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
)

type SpendAccountAction struct {
	AccountID string `json:"account_id"`
	*bc.AssetAmount
}

func (s *SpendAccountAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type      string `json:"type"`
		AccountID string `json:"account_id"`
		*bc.AssetAmount
	}{
		Type:        "spend_account",
		AccountID:   s.AccountID,
		AssetAmount: s.AssetAmount,
	})
}

type ControlAddressAction struct {
	Address string `json:"address"`
	*bc.AssetAmount
}

func (c *ControlAddressAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type    string `json:"type"`
		Address string `json:"address"`
		*bc.AssetAmount
	}{
		Type:        "control_address",
		Address:     c.Address,
		AssetAmount: c.AssetAmount,
	})
}

func (n *Node) BatchSendBTM(accountID, password string, outputs map[string]uint64) error {
	totalBTM := uint64(10000000)
	actions := []interface{}{}
	for address, amount := range outputs {
		actions = append(actions, &ControlAddressAction{
			Address:     address,
			AssetAmount: &bc.AssetAmount{AssetId: consensus.BTMAssetID, Amount: amount},
		})
		totalBTM += amount
	}

	actions = append(actions, &SpendAccountAction{
		AccountID:   accountID,
		AssetAmount: &bc.AssetAmount{AssetId: consensus.BTMAssetID, Amount: totalBTM},
	})

	tpl, err := n.buildTx(actions)
	if err != nil {
		return err
	}

	tpl, err = n.signTx(tpl, password)
	if err != nil {
		return err
	}

	_, err = n.SubmitTx(tpl.Transaction)
	return err
}

type buildTxReq struct {
	Actions []interface{} `json:"actions"`
}

func (n *Node) buildTx(actions []interface{}) (*txbuilder.Template, error) {
	url := "/build-transaction"
	payload, err := json.Marshal(&buildTxReq{Actions: actions})
	if err != nil {
		return nil, errors.Wrap(err, "Marshal spend request")
	}

	result := &txbuilder.Template{}
	return result, n.request(url, payload, result)
}

type signTxReq struct {
	Tx       *txbuilder.Template `json:"transaction"`
	Password string              `json:"password"`
}

type signTxResp struct {
	Tx           *txbuilder.Template `json:"transaction"`
	SignComplete bool                `json:"sign_complete"`
}

func (n *Node) signTx(tpl *txbuilder.Template, password string) (*txbuilder.Template, error) {
	url := "/sign-transaction"
	payload, err := json.Marshal(&signTxReq{Tx: tpl, Password: password})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	resp := &signTxResp{}
	if err := n.request(url, payload, resp); err != nil {
		return nil, err
	}

	if !resp.SignComplete {
		return nil, errors.New("sign fail")
	}

	return resp.Tx, nil
}

type submitTxReq struct {
	Tx *types.Tx `json:"raw_transaction"`
}

type submitTxResp struct {
	TxID string `json:"tx_id"`
}

func (n *Node) SubmitTx(tx *types.Tx) (string, error) {
	url := "/submit-transaction"
	payload, err := json.Marshal(submitTxReq{Tx: tx})
	if err != nil {
		return "", errors.Wrap(err, "json marshal")
	}

	res := &submitTxResp{}
	return res.TxID, n.request(url, payload, res)
}
