package api

import (
	"context"

	"github.com/bytom/vapor/blockchain/txbuilder"
)

type AccountFilter struct {
	AccountID    string `json:"account_id"`
	AccountAlias string `json:"account_alias"`
}

func (a *API) createAccountReceiver(ctx context.Context, ins AccountFilter) Response {
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
