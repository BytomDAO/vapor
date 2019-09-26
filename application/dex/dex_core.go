package dex

import (
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

type DexCore struct {}

func (d *DexCore) ChainStatus()(uint64, *bc.Hash) {
	return 0, nil
}

func (d *DexCore) ValidateBlock(block *bc.Block) error{
	return nil
}

func (d *DexCore) ValidateTxs(txs []*bc.Tx) error {
	return nil
}

func (d *DexCore) ApplyBlock(block *types.Block) error {
	return nil
}

func (d *DexCore) DetachBlock(block *types.Block) error {
	return nil
}

func (d *DexCore) DBeforeProposalBlock(txs []*types.Tx, num int) ([]*types.Tx,error) {
	return nil, nil
}

func (d *DexCore) IsDust(tx *types.Tx) bool {
	return false
}
