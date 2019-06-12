package api

import (
	chainjson "github.com/vapor/encoding/json"
)

func (a *API) getConsensusNodes(req struct {
	BlockHash chainjson.HexBytes `json:"block_hash"`
}) Response {
	hash := hexBytesToHash(req.BlockHash)
	consensusNodes, err := a.chain.GetConsensusNodes(&hash)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(consensusNodes)
}
