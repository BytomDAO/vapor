package database

import (
	"github.com/golang/groupcache/lru"
)

const maxAssetCached = 1024

type AssetCache struct {
	lruCache *lru.Cache
}

func NewAssetCache() *AssetCache {
	return &AssetCache{lruCache: lru.New(maxAssetCached)}
}

func (a *AssetCache) Add() {
}

func (a *AssetCache) Get() {
}

func (a *AssetCache) Remove() {
}
