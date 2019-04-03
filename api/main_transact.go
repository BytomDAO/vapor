package api

import (
	"github.com/vapor/claim/rpc"
	maintx "github.com/vapor/claim/rpc/bytom"
	chainjson "github.com/vapor/encoding/json"
)

type mainTxResp struct {
	Tx chainjson.HexBytes `json:"tx"`
}

func (a *API) buildMainChainTxForContract(ins rpc.MainTxParam) Response {
	main := &maintx.BytomMainTx{
		MainTxParam: ins,
	}

	resp, err := main.BuildMainChainTxForContract()
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(&mainTxResp{Tx: resp})
}

func (a *API) buildMainChainTx(ins rpc.MainTxParam) Response {
	main := &maintx.BytomMainTx{
		MainTxParam: ins,
	}

	resp, err := main.BuildMainChainTx()
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(&mainTxResp{Tx: resp})
}

func (a *API) signWithKey(ins rpc.MainTxSignParam) Response {
	sign := maintx.BytomMainSign{
		MainTxSignParam: ins,
	}

	resp, err := sign.SignWithKey()
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(resp)
}
