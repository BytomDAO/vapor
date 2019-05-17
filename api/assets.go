package api

// POST /update-asset-alias
func (a *API) updateAssetAlias(updateAlias struct {
	ID       string `json:"id"`
	NewAlias string `json:"alias"`
}) Response {
	if err := a.wallet.AssetReg.UpdateAssetAlias(updateAlias.ID, updateAlias.NewAlias); err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(nil)
}
