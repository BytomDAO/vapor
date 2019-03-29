package rpc

import (
	"context"

	chainjson "github.com/vapor/encoding/json"
)

type ClaimTxParam struct {
	Password     string               `json:"password"`
	RawTx        string               `json:"raw_transaction"`
	BlockHeader  string               `json:"block_header"`
	TxHashes     []chainjson.HexBytes `json:"tx_hashes"`
	StatusHashes []chainjson.HexBytes `json:"status_hashes"`
	Flags        []uint32             `json:"flags"`
	MatchedTxIDs []chainjson.HexBytes `json:"matched_tx_ids"`
	ClaimScript  chainjson.HexBytes   `json:"claim_script"`
}

type ClaimTx interface {
	ClaimPeginTx(ctx context.Context) (interface{}, error)
	ClaimContractPeginTx(ctx context.Context) (interface{}, error)
}
