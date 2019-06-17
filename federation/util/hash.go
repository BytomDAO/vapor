package util

import (
	"encoding/hex"

	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
)

func stringToAssetID(s string) (*bc.AssetID, error) {
	h, err := hex.DecodeString(s)
	if err != nil {
		return nil, errors.Wrap(err, "decode asset string")
	}

	var b [32]byte
	copy(b[:], h)
	assetID := bc.NewAssetID(b)
	return &assetID, nil
}
