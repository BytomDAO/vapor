package common

import "github.com/vapor/protocol/bc"

type MovUtxo struct {
	SourceID       *bc.Hash
	SourcePos      uint64
	Amount         uint64
	ControlProgram []byte
}

type Order struct {
	FromAssetID *bc.AssetID
	ToAssetID   *bc.AssetID
	Utxo        *MovUtxo
	Rate        float64
}

type TradePair struct {
	FromAssetID *bc.AssetID
	ToAssetID   *bc.AssetID
	Count       int
}

type MovDatabaseState struct {
	Height uint64
	Hash   *bc.Hash
}
