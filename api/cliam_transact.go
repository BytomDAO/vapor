package api

import (
	"context"

	"github.com/vapor/claim/rpc"
	claimtx "github.com/vapor/claim/rpc/bytom"
)

func (a *API) claimPeginTx(ctx context.Context, ins rpc.ClaimTxParam) Response {

	c := &claimtx.BytomClaimTx{
		ClaimTxParam: ins,
		Wallet:       a.wallet,
		Chain:        a.chain,
	}
	resp, err := c.ClaimPeginTx(ctx)

	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(resp)
}

func (a *API) claimContractPeginTx(ctx context.Context, ins rpc.ClaimTxParam) Response {
	c := &claimtx.BytomClaimTx{
		ClaimTxParam: ins,
		Wallet:       a.wallet,
		Chain:        a.chain,
	}
	resp, err := c.ClaimContractPeginTx(ctx)

	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(resp)
}
