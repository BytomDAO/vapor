package orm

import "github.com/vapor/protocol/bc"

type BlockStoreState struct {
	StoreKey string `gorm:"primary_key"`
	Height   uint64
	Hash     string
}

type BlockHeader struct {
	Height                 uint64
	BlockHash              string
	Version                uint64
	PreviousBlockHash      string
	Timestamp              uint64
	TransactionsMerkleRoot string
	TransactionStatusHash  string
}

func stringToHash(str string) (*bc.Hash, error) {
	hash := &bc.Hash{}
	if err := hash.UnmarshalText([]byte(str)); err != nil {
		return nil, err
	}
	return hash, nil
}

func (b *BlockHeader) PreBlockHash() (*bc.Hash, error) {
	return stringToHash(b.PreviousBlockHash)
}

func (b *BlockHeader) MerkleRoot() (*bc.Hash, error) {
	return stringToHash(b.TransactionsMerkleRoot)
}

func (b *BlockHeader) StatusHash() (*bc.Hash, error) {
	return stringToHash(b.TransactionStatusHash)
}
