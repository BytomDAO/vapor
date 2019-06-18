package service

import (
	"github.com/jinzhu/gorm"

	"github.com/vapor/errors"
	"github.com/vapor/federation/database"
	"github.com/vapor/federation/database/orm"
)

type AssetKeeper struct {
	db         *gorm.DB
	assetCache *database.AssetCache
}

func NewAssetKeeper(db *gorm.DB) *AssetKeeper {
	return &AssetKeeper{
		db:         db,
		assetCache: database.NewAssetCache(),
	}
}

func (a *AssetKeeper) Get(assetID string) (*orm.Asset, error) {
	if asset := a.assetCache.Get(assetID); asset != nil {
		return asset, nil
	}

	asset := &orm.Asset{AssetID: assetID}
	if err := a.db.Where(asset).First(asset).Error; err != nil {
		return nil, errors.Wrap(err, "asset not found in memory and mysql")
	}

	a.assetCache.Add(assetID, asset)
	return asset, nil
}

func (a *AssetKeeper) Add(asset *orm.Asset) error {
	if err := a.db.Create(asset).Error; err != nil {
		return err
	}

	a.assetCache.Add(asset.AssetID, asset)
	return nil
}
