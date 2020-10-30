package ed25519

import (
	"encoding/hex"
	"strings"

	"github.com/bytom/vapor/common"
)

const verifyCacheSize = 8192

var cache *common.Cache

// InitCache will enable the ed25519 cached
func InitCache() {
	cache = common.NewCache(verifyCacheSize)
}

func cacheKey(publicKey PublicKey, message, sig []byte) string {
	return strings.Join([]string{hex.EncodeToString(publicKey), hex.EncodeToString(message), hex.EncodeToString(sig)}, ":")
}

func checkVerifyCache(publicKey PublicKey, message, sig []byte) bool {
	if cache == nil {
		return false
	}

	_, isVerified := cache.Get(cacheKey(publicKey, message, sig))
	return isVerified
}

func saveVerifyCache(publicKey PublicKey, message, sig []byte) {
	if cache != nil {
		cache.Add(cacheKey(publicKey, message, sig), true)
	}
}
