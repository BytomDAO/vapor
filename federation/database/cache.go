package database

import (
	"github.com/golang/groupcache/lru"
)

const maxAssetCached = 1024

type AssetCache struct {
	lruCache *lru.Cache
}

func NewAssetCache() *AssetCache {
	return &AssetCache{}
}
