package api

import (
	"context"

	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/claim/rpc"
	chainjson "github.com/vapor/encoding/json"
)

func (a *API) createAccountReceiver(ctx context.Context, ins struct {
	AccountID    string `json:"account_id"`
	AccountAlias string `json:"account_alias"`
}) Response {
	accountID := ins.AccountID
	if ins.AccountAlias != "" {
		account, err := a.wallet.AccountMgr.FindByAlias(ins.AccountAlias)
		if err != nil {
			return NewErrorResponse(err)
		}

		accountID = account.ID
	}

	program, err := a.wallet.AccountMgr.CreateAddress(accountID, false)
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(&txbuilder.Receiver{
		ControlProgram: program.ControlProgram,
		Address:        program.Address,
	})
}

type fundingResp struct {
	MainchainAddress string             `json:"mainchain_address"`
	ControlProgram   chainjson.HexBytes `json:"control_program,omitempty"`
	ClaimScript      chainjson.HexBytes `json:"claim_script"`
}

func (a *API) getPeginAddress(ctx context.Context, ins rpc.ClaimArgs) Response {

	pegin := &rpc.BytomPeginRpc{
		ClaimArgs: ins,
		Wallet:    a.wallet,
	}

	resp, err := pegin.GetPeginAddress()
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(resp)
}

func (a *API) getPeginContractAddress(ctx context.Context, ins struct {
	AccountID    string `json:"account_id"`
	AccountAlias string `json:"account_alias"`
}) Response {
	pegin := &rpc.BytomPeginRpc{
		ClaimArgs: ins,
		Wallet:    a.wallet,
	}
	resp, err := pegin.GetPeginContractAddress()
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(resp)
}
