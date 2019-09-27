package protocol

import (
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

// startHeight mov protocol startup height.
const startHeight = 0

type combination interface {
	ApplyBlock(block *types.Block) error
	BeforeProposalBlock(txs []*types.Tx) ([]*types.Tx, error)
}

type MOV struct {
	combination combination
}

func NewMOV() *MOV {
	return &MOV{}
}

func (m MOV) ApplyBlock(block *types.Block) error {
	return m.combination.ApplyBlock(block)
}

func (m MOV) BeforeProposalBlock(txs []*types.Tx) ([]*types.Tx, error) {
	return m.combination.BeforeProposalBlock(txs)
}

func (m MOV) ChainStatus() (uint64, *bc.Hash) {
	return 0, nil
}

func (m MOV) DetachBlock(block *types.Block) error {
	return nil
}

func (m MOV) IsDust(tx *types.Tx) bool {
	return false
}

func (m MOV) Name() string {
	return "MOV"
}

func (m MOV) ValidateBlock(block *bc.Block) error {
	return nil
}

func (m MOV) ValidateTxs(txs []*bc.Tx) error {
	return nil
}

func (m MOV) Status() (uint64, *bc.Hash){
	return 0,nil
}

func (m MOV) SyncStatus() error {
	return nil
}
