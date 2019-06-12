package database

import (
	"github.com/golang/groupcache/lru"

	"github.com/vapor/federation/database/orm"
)

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
