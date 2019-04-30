package orm

import (
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

type BlockStoreState struct {
	StoreKey string `gorm:"primary_key"`
	Height   uint64
	Hash     string
}

type BlockHeader struct {
	ID                     uint   `gorm:"AUTO_INCREMENT"`
	BlockHash              string `gorm:"primary_key"`
	Height                 uint64
	Version                uint64
	PreviousBlockHash      string
	Timestamp              uint64
	TransactionsMerkleRoot string
	TransactionStatusHash  string

	tx *Transaction `gorm:"FOREIGNKEY:ID;AssociationForeignKey:BlockHeaderID"`
}

func stringToHash(str string) (*bc.Hash, error) {
	hash := &bc.Hash{}
	if err := hash.UnmarshalText([]byte(str)); err != nil {
		return nil, err
	}
	return hash, nil
}

func (bh *BlockHeader) PreBlockHash() (*bc.Hash, error) {
	return stringToHash(bh.PreviousBlockHash)
}

func (bh *BlockHeader) MerkleRoot() (*bc.Hash, error) {
	return stringToHash(bh.TransactionsMerkleRoot)
}

func (bh *BlockHeader) StatusHash() (*bc.Hash, error) {
	return stringToHash(bh.TransactionStatusHash)
}

func (bh *BlockHeader) BcBlockHeader() (*types.BlockHeader, error) {
	previousBlockHash, err := bh.PreBlockHash()
	if err != nil {
		return nil, err
	}

	transactionsMerkleRoot, err := bh.MerkleRoot()
	if err != nil {
		return nil, err
	}
	transactionStatusHash, err := bh.StatusHash()
	if err != nil {
		return nil, err
	}

	return &types.BlockHeader{
		Version:           bh.Version,
		Height:            bh.Height,
		PreviousBlockHash: *previousBlockHash,
		Timestamp:         bh.Timestamp,
		BlockCommitment: types.BlockCommitment{
			TransactionsMerkleRoot: *transactionsMerkleRoot,
			TransactionStatusHash:  *transactionStatusHash,
		},
	}, nil
}
