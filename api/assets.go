package api

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/vapor/asset"
	"github.com/bytom/vapor/crypto/ed25519/chainkd"
)

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

func (a *API) createAsset(ctx context.Context, ins struct {
	Alias      string                 `json:"alias"`
	RootXPubs  []chainkd.XPub         `json:"root_xpubs"`
	Quorum     int                    `json:"quorum"`
	Definition map[string]interface{} `json:"definition"`
}) Response {
	rawAsset, err := a.wallet.AssetReg.Define(
		ins.RootXPubs,
		ins.Quorum,
		ins.Definition,
		strings.ToUpper(strings.TrimSpace(ins.Alias)),
	)
	if err != nil {
		return NewErrorResponse(err)
	}

	annotatedAsset, err := asset.Annotated(rawAsset)
	if err != nil {
		return NewErrorResponse(err)
	}

	log.WithField("asset ID", annotatedAsset.ID.String()).Info("Created asset")

	return NewSuccessResponse(annotatedAsset)
}
