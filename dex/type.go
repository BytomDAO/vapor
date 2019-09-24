package dex

import "github.com/vapor/protocol/bc"

type DexUtxo struct {
	SourceID       bc.Hash
	AssetID        bc.AssetID
	Amount         uint64
	SourcePos      uint64
	ControlProgram []byte
}

type Order struct {
	ToAssetID bc.AssetID
	Rate      float64
	Utxo      DexUtxo
}

type TradePair struct {
	FromAssetID *bc.AssetID
	ToAssetID   *bc.AssetID
	Count       uint64
}

type MatchState struct {
	Height uint64
	Hash   *bc.Hash
}
