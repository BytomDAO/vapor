package api

import (
	"context"
	"strconv"

	chainjson "github.com/bytom/vapor/encoding/json"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/event"
	"github.com/bytom/vapor/protocol/bc/types"
)

type CoinbaseArbitrary struct {
	Arbitrary chainjson.HexBytes `json:"arbitrary"`
}

func (a *API) getCoinbaseArbitrary() Response {
	arbitrary := a.wallet.AccountMgr.GetCoinbaseArbitrary()
	resp := &CoinbaseArbitrary{
		Arbitrary: arbitrary,
	}
	return NewSuccessResponse(resp)
}

// setCoinbaseArbitrary add arbitary data to the reserved coinbase data.
// check function createCoinbaseTx in mining/mining.go for detail.
// arbitraryLenLimit is 107 and can be calculated by:
// 	maxHeight := ^uint64(0)
// 	reserved := append([]byte{0x00}, []byte(strconv.FormatUint(maxHeight, 10))...)
// 	arbitraryLenLimit := consensus.CoinbaseArbitrarySizeLimit - len(reserved)
func (a *API) setCoinbaseArbitrary(ctx context.Context, req CoinbaseArbitrary) Response {
	arbitraryLenLimit := 107
	if len(req.Arbitrary) > arbitraryLenLimit {
		err := errors.New("Arbitrary exceeds limit: " + strconv.FormatUint(uint64(arbitraryLenLimit), 10))
		return NewErrorResponse(err)
	}
	a.wallet.AccountMgr.SetCoinbaseArbitrary(req.Arbitrary)
	return a.getCoinbaseArbitrary()
}

// SubmitBlockReq is req struct for submit-block API
type SubmitBlockReq struct {
	Block *types.Block `json:"raw_block"`
}

// submitBlock trys to submit a raw block to the chain
func (a *API) submitBlock(ctx context.Context, req *SubmitBlockReq) Response {
	isOrphan, err := a.chain.ProcessBlock(req.Block)
	if err != nil {
		return NewErrorResponse(err)
	}

	if isOrphan {
		return NewErrorResponse(errors.New("block submitted is orphan"))
	}

	if err = a.eventDispatcher.Post(event.NewProposedBlockEvent{Block: *req.Block}); err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(true)
}

func (a *API) setMining(in struct {
	IsMining bool `json:"is_mining"`
}) Response {
	if in.IsMining {
		if _, err := a.wallet.AccountMgr.GetMiningAddress(); err != nil {
			return NewErrorResponse(errors.New("Mining address does not exist"))
		}
		return a.startMining()
	}
	return a.stopMining()
}

func (a *API) startMining() Response {
	a.blockProposer.Start()
	if !a.IsMining() {
		return NewErrorResponse(errors.New("Failed to start mining"))
	}
	return NewSuccessResponse("")
}

func (a *API) stopMining() Response {
	a.blockProposer.Stop()
	if a.IsMining() {
		return NewErrorResponse(errors.New("Failed to stop mining"))
	}
	return NewSuccessResponse("")
}
