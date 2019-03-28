package rpc

import (
	chainjson "github.com/vapor/encoding/json"
)

type MainTxSignParam struct {
	Xprv        string             `json:"xprv"`
	XPub        string             `json:"xpub"`
	Txs         chainjson.HexBytes `json:"transaction"`
	ClaimScript chainjson.HexBytes `json:"claim_script"`
}

type MainTxParam struct {
	Utxo           []byte             `json:"utxo"`
	Tx             string             `json:"raw_transaction"`
	Pubs           []string           `json:"pubs"`
	ControlProgram string             `json:"control_program"`
	ClaimScript    chainjson.HexBytes `json:"claim_script"`
}
