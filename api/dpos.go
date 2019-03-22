package api

import (
	"context"

	dpos "github.com/vapor/consensus/consensus/dpos"
)

func (a *API) listDelegates(ctx context.Context) Response {
	return NewSuccessResponse(dpos.GDpos.ListDelegates())
}

func (a *API) getDelegateVotes(ctx context.Context, ins struct {
	DelegateAddress string `json:"delegate_address"`
}) Response {
	votes := map[string]uint64{"votes": dpos.GDpos.GetDelegateVotes(ins.DelegateAddress)}
	return NewSuccessResponse(votes)
}

func (a *API) listVotedDelegates(ctx context.Context, ins struct {
	Voter string `json:"voter"`
}) Response {
	delegates := make(map[string]string)
	for _, delegate := range dpos.GDpos.GetVotedDelegates(ins.Voter) {
		delegates[dpos.GDpos.GetDelegateName(delegate)] = delegate
	}
	return NewSuccessResponse(delegates)
}

func (a *API) listReceivedVotes(ctx context.Context, ins struct {
	DelegateAddress string `json:"delegate_address"`
}) Response {
	return NewSuccessResponse(dpos.GDpos.GetDelegateVoters(ins.DelegateAddress))
}

func (a *API) getAddressBalance(ctx context.Context, ins struct {
	Address string `json:"address"`
}) Response {
	return NewSuccessResponse(dpos.GDpos.GetAddressBalance(ins.Address))
}
