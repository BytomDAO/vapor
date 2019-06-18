package database

import (
	"github.com/golang/groupcache/lru"
	"github.com/jinzhu/gorm"

	"github.com/vapor/errors"
	"github.com/vapor/federation/database/orm"
)

const maxAssetCached = 1024

type AssetKeeper struct {
	cache *lru.Cache
	db    *gorm.DB
}

func NewAssetKeeper(db *gorm.DB) *AssetKeeper {
	return &AssetKeeper{
		cache: lru.New(maxAssetCached),
		db:    db,
	}
}

func (a *AssetKeeper) GetByOrmID(id uint64) (*orm.Asset, error) {
	asset := &orm.Asset{ID: id}
	if err := a.db.Where(asset).First(asset).Error; err != nil {
		return nil, errors.Wrap(err, "asset not found by orm id")
	}

	return asset, nil
}

func (a *AssetKeeper) Get(assetID string) (*orm.Asset, error) {
	if v, ok := a.cache.Get(assetID); ok {
		return v.(*orm.Asset), nil
	}

	asset := &orm.Asset{AssetID: assetID}
	if err := a.db.Where(asset).First(asset).Error; err != nil {
		return nil, errors.Wrap(err, "asset not found in memory and mysql")
	}

	a.cache.Add(assetID, asset)
	return asset, nil
}

func (a *AssetKeeper) Add(asset *orm.Asset) error {
	if err := a.db.Create(asset).Error; err != nil {
		return err
	}

	a.cache.Add(asset.AssetID, asset)
	return nil
}
