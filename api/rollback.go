package api

func (a *API) rollbackBlock(req struct{
	BlockHeight uint64 `json:"block_height"`
}) Response {
	for a.chain.BestBlockHeight() > req.BlockHeight {
		if err := a.chain.DetachLast(); err != nil {
			return NewErrorResponse(err)
		}
	}
	return NewSuccessResponse(nil)
}
