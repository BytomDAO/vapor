package database

import (
	"fmt"

	"github.com/golang/groupcache/lru"
	"github.com/jinzhu/gorm"

	"github.com/vapor/errors"
	"github.com/vapor/toolbar/federation/database/orm"
)

const (
	maxAssetCached = 1024

	ormIDPrefix   = "ormID"
	assetIDPrefix = "assetID"
)

func fmtOrmIDKey(ormID uint64) string {
	return fmt.Sprintf("%s:%d", ormIDPrefix, ormID)
}

func fmtAssetIDKey(assetID string) string {
	return fmt.Sprintf("%s:%s", assetIDPrefix, assetID)
}

type AssetStore struct {
	cache *lru.Cache
	db    *gorm.DB
}

func NewAssetStore(db *gorm.DB) *AssetStore {
	return &AssetStore{
		cache: lru.New(maxAssetCached),
		db:    db,
	}
}

func (a *AssetStore) GetByOrmID(ormID uint64) (*orm.Asset, error) {
	if v, ok := a.cache.Get(fmtOrmIDKey(ormID)); ok {
		return v.(*orm.Asset), nil
	}

	asset := &orm.Asset{ID: ormID}
	if err := a.db.Where(asset).First(asset).Error; err != nil {
		return nil, errors.Wrap(err, "asset not found by orm id")
	}

	a.cache.Add(fmtOrmIDKey(asset.ID), asset)
	a.cache.Add(fmtAssetIDKey(asset.AssetID), asset)
	return asset, nil
}

func (a *AssetStore) GetByAssetID(assetID string) (*orm.Asset, error) {
	if v, ok := a.cache.Get(fmtAssetIDKey(assetID)); ok {
		return v.(*orm.Asset), nil
	}

	asset := &orm.Asset{AssetID: assetID}
	if err := a.db.Where(asset).First(asset).Error; err != nil {
		return nil, errors.Wrap(err, "asset not found in memory and mysql")
	}

	a.cache.Add(fmtOrmIDKey(asset.ID), asset)
	a.cache.Add(fmtAssetIDKey(asset.AssetID), asset)
	return asset, nil
}
