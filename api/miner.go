package api

import (
	"context"
	"strconv"

	chainjson "github.com/vapor/encoding/json"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

// BlockHeaderJSON struct provides support for get work in json format, when it also follows
// BlockHeader structure
type BlockHeaderJSON struct {
	Version           uint64                 `json:"version"`             // The version of the block.
	Height            uint64                 `json:"height"`              // The height of the block.
	PreviousBlockHash bc.Hash                `json:"previous_block_hash"` // The hash of the previous block.
	Timestamp         uint64                 `json:"timestamp"`           // The time of the block in seconds.
	Nonce             uint64                 `json:"nonce"`               // Nonce used to generate the block.
	Bits              uint64                 `json:"bits"`                // Difficulty target for the block.
	BlockCommitment   *types.BlockCommitment `json:"block_commitment"`    // Block commitment
}

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

	blockHash := req.Block.BlockHeader.Hash()
	a.newBlockCh <- &blockHash
	return NewSuccessResponse(true)
}

// SubmitWorkReq is req struct for submit-work API
type SubmitWorkReq struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
}

// SubmitWorkJSONReq is req struct for submit-work-json API
type SubmitWorkJSONReq struct {
	BlockHeader *BlockHeaderJSON `json:"block_header"`
}

// GetWorkResp is resp struct for get-work API
type GetWorkResp struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
	Seed        *bc.Hash           `json:"seed"`
}

// GetWorkJSONResp is resp struct for get-work-json API
type GetWorkJSONResp struct {
	BlockHeader *BlockHeaderJSON `json:"block_header"`
	Seed        *bc.Hash         `json:"seed"`
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
	//a.cpuMiner.Start()
	a.miner.Start()
	if !a.IsMining() {
		return NewErrorResponse(errors.New("Failed to start mining"))
	}
	return NewSuccessResponse("")
}

func (a *API) stopMining() Response {
	//a.cpuMiner.Stop()
	a.miner.Stop()
	if a.IsMining() {
		return NewErrorResponse(errors.New("Failed to stop mining"))
	}
	return NewSuccessResponse("")
}
