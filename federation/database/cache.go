package database

import (
	// use hashicorp/golang-lru instead of golang/groupcache/lru for thread safety
	lru "github.com/hashicorp/golang-lru"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/federation/database/orm"
)

const maxAssetCached = 1024

type AssetCache struct {
	lruCache *lru.Cache
}

func NewAssetCache() *AssetCache {
	cache, err := lru.New(maxAssetCached)
	if err != nil {
		log.Fatalf("lruCache init error: %v", err)
	}

	return &AssetCache{lruCache: cache}
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
