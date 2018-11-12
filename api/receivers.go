package api

import (
	"context"

	"github.com/bytom/blockchain/txbuilder"
	chainjson "github.com/bytom/encoding/json"
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
	ClaimScript      chainjson.HexBytes `json:"claim_script"`
}

func (a *API) getPeginAddress(ctx context.Context, ins struct {
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

	mainchainAddress, claimScript, err := a.wallet.AccountMgr.CreatePeginAddress(accountID, false)
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(fundingResp{
		MainchainAddress: mainchainAddress,
		ClaimScript:      claimScript,
	})
}
