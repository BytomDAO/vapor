package common

import "github.com/vapor/protocol/bc"

type DexUtxo struct {
	SourceID       *bc.Hash
	SourcePos      uint64
	Amount         uint64
	ControlProgram []byte
}

type Order struct {
	FromAssetID *bc.AssetID
	ToAssetID   *bc.AssetID
	Utxo        *DexUtxo
	Rate        float64
}

type TradePair struct {
	FromAssetID *bc.AssetID
	ToAssetID   *bc.AssetID
	Count       uint64
}

type DexDatabaseState struct {
	Height uint64
	Hash   *bc.Hash
}