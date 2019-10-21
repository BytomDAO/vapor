package protocol

import (
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	protocolName = "MOV"
)

type matchEnginer interface {
	ApplyBlock(block *types.Block) error
	BeforeProposalBlock(capacity int) ([]*types.Tx, error)
	ChainStatus() (uint64, *bc.Hash, error)
	DetachBlock(block *types.Block) error
	IsDust(tx *types.Tx) bool
	ValidateBlock(block *types.Block) error
	ValidateTxs(txs []*types.Tx) error
}

type MOV struct {
	engine matchEnginer
}

func NewMOV() *MOV {
	return &MOV{}
}

func (m MOV) ApplyBlock(block *types.Block) error {
	return m.engine.ApplyBlock(block)
}

func (m MOV) BeforeProposalBlock(capacity int) ([]*types.Tx, error) {
	return m.engine.BeforeProposalBlock(capacity)
}

func (m MOV) ChainStatus() (uint64, *bc.Hash, error) {
	return m.engine.ChainStatus()
}

func (m MOV) DetachBlock(block *types.Block) error {
	return m.engine.DetachBlock(block)
}

func (m MOV) IsDust(tx *types.Tx) bool {
	return m.engine.IsDust(tx)
}

func (m MOV) Name() string {
	return protocolName
}

func (m MOV) ValidateBlock(block *types.Block) error {
	return m.engine.ValidateBlock(block)
}

func (m MOV) ValidateTxs(txs []*types.Tx) error {
	return m.engine.ValidateTxs(txs)
}
