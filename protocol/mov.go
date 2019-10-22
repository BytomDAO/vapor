package protocol

import (
	"github.com/vapor/application/mov"
	"github.com/vapor/config"
	"github.com/vapor/consensus"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	protocolName = "MOV"
)

type movCore interface {
	ApplyBlock(block *types.Block) error
	BeforeProposalBlock(capacity int) ([]*types.Tx, error)
	ChainStatus() (uint64, *bc.Hash, error)
	DetachBlock(block *types.Block) error
	IsDust(tx *types.Tx) bool
	ValidateBlock(block *types.Block) error
	ValidateTxs(txs []*types.Tx) error
}

type MOV struct {
	core movCore
}

func NewMOV(db dbm.DB, startPoint consensus.Checkpoint) (*MOV, error) {
	if startPoint.Height == 0 {
		startPoint.Hash = config.GenesisBlock().Hash()
	}

	movCore, err := mov.NewMovCore(db, startPoint.Height, &startPoint.Hash)
	if err != nil {
		return nil, errors.Wrap(err, "failed on create mov core")
	}

	return &MOV{
		core: movCore,
	}, nil
}

func (m MOV) ApplyBlock(block *types.Block) error {
	return m.core.ApplyBlock(block)
}

func (m MOV) BeforeProposalBlock(capacity int) ([]*types.Tx, error) {
	return m.core.BeforeProposalBlock(capacity)
}

func (m MOV) ChainStatus() (uint64, *bc.Hash, error) {
	return m.core.ChainStatus()
}

func (m MOV) DetachBlock(block *types.Block) error {
	return m.core.DetachBlock(block)
}

func (m MOV) IsDust(tx *types.Tx) bool {
	return m.core.IsDust(tx)
}

func (m MOV) Name() string {
	return protocolName
}

func (m MOV) ValidateBlock(block *types.Block) error {
	return m.core.ValidateBlock(block)
}

func (m MOV) ValidateTxs(txs []*types.Tx) error {
	return m.core.ValidateTxs(txs)
}
