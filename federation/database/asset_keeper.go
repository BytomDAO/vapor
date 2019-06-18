package database

import (
	"github.com/golang/groupcache/lru"
	"github.com/jinzhu/gorm"

	"github.com/vapor/errors"
	"github.com/vapor/federation/database/orm"
)

// TODO:
type AssetKeeper struct {
	db         *gorm.DB
	assetCache *AssetCache
}

func NewAssetKeeper(db *gorm.DB) *AssetKeeper {
	return &AssetKeeper{
		db:         db,
		assetCache: NewAssetCache(),
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

const maxAssetCached = 1024

type AssetCache struct {
	lruCache *lru.Cache
}

func NewAssetCache() *AssetCache {
	return &AssetCache{lruCache: lru.New(maxAssetCached)}
}

func (a *AssetCache) Add(assetID string, asset *orm.Asset) {
	a.lruCache.Add(assetID, asset)
}

func (a *AssetCache) Get(assetID string) *orm.Asset {
	if v, ok := a.lruCache.Get(assetID); ok {
		return v.(*orm.Asset)
	}

	return nil
}

func (a *AssetCache) Remove(assetID string) {
	a.lruCache.Remove(assetID)
}
